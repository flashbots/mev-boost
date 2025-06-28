import React, { useState, useEffect } from 'react';
import { 
  History, 
  ExternalLink, 
  Copy, 
  CheckCircle, 
  XCircle, 
  Clock,
  Search,
  Filter,
  Download
} from 'lucide-react';

interface DeploymentRecord {
  id: string;
  timestamp: Date;
  contractName: string;
  sourceCodeHash: string;
  totalChains: number;
  successfulChains: number;
  failedChains: number;
  deployments: Array<{
    chainId: number;
    chainName: string;
    status: 'success' | 'failed';
    contractAddress?: string;
    transactionHash?: string;
    gasUsed?: string;
    error?: string;
  }>;
}

export const DeploymentHistory: React.FC = () => {
  const [deploymentHistory, setDeploymentHistory] = useState<DeploymentRecord[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'success' | 'failed'>('all');
  const [selectedRecord, setSelectedRecord] = useState<DeploymentRecord | null>(null);

  useEffect(() => {
    loadDeploymentHistory();
  }, []);

  const loadDeploymentHistory = () => {
    // Mock deployment history
    const mockHistory: DeploymentRecord[] = [
      {
        id: '1',
        timestamp: new Date(Date.now() - 86400000), // 1 day ago
        contractName: 'MyToken',
        sourceCodeHash: '0xabc123...',
        totalChains: 15,
        successfulChains: 12,
        failedChains: 3,
        deployments: [
          {
            chainId: 1,
            chainName: 'Ethereum Mainnet',
            status: 'success',
            contractAddress: '0x1234567890123456789012345678901234567890',
            transactionHash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
            gasUsed: '2100000'
          },
          {
            chainId: 137,
            chainName: 'Polygon Mainnet',
            status: 'success',
            contractAddress: '0x0987654321098765432109876543210987654321',
            transactionHash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
            gasUsed: '1800000'
          },
          {
            chainId: 56,
            chainName: 'BNB Smart Chain',
            status: 'failed',
            error: 'Insufficient gas'
          }
        ]
      },
      {
        id: '2',
        timestamp: new Date(Date.now() - 172800000), // 2 days ago
        contractName: 'NFTCollection',
        sourceCodeHash: '0xdef456...',
        totalChains: 8,
        successfulChains: 8,
        failedChains: 0,
        deployments: [
          {
            chainId: 1,
            chainName: 'Ethereum Mainnet',
            status: 'success',
            contractAddress: '0x2345678901234567890123456789012345678901',
            transactionHash: '0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890a',
            gasUsed: '3200000'
          }
        ]
      }
    ];

    setDeploymentHistory(mockHistory);
  };

  const filteredHistory = deploymentHistory.filter(record => {
    const matchesSearch = record.contractName.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesStatus = statusFilter === 'all' || 
      (statusFilter === 'success' && record.failedChains === 0) ||
      (statusFilter === 'failed' && record.failedChains > 0);
    
    return matchesSearch && matchesStatus;
  });

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const exportDeploymentData = (record: DeploymentRecord) => {
    const data = {
      contractName: record.contractName,
      timestamp: record.timestamp.toISOString(),
      deployments: record.deployments
    };
    
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${record.contractName}-deployment-${record.id}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'failed':
        return <XCircle className="w-4 h-4 text-red-500" />;
      default:
        return <Clock className="w-4 h-4 text-gray-400" />;
    }
  };

  if (selectedRecord) {
    return (
      <div className="max-w-6xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <button
            onClick={() => setSelectedRecord(null)}
            className="btn-secondary flex items-center space-x-2"
          >
            <span>← Back to History</span>
          </button>
          
          <button
            onClick={() => exportDeploymentData(selectedRecord)}
            className="btn-primary flex items-center space-x-2"
          >
            <Download className="w-4 h-4" />
            <span>Export Data</span>
          </button>
        </div>

        <div className="card p-6 mb-6">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{selectedRecord.contractName}</h1>
              <p className="text-gray-600">
                Deployed on {selectedRecord.timestamp.toLocaleDateString()} at{' '}
                {selectedRecord.timestamp.toLocaleTimeString()}
              </p>
            </div>
            
            <div className="text-right">
              <div className="text-sm text-gray-500">Success Rate</div>
              <div className="text-2xl font-bold text-green-600">
                {Math.round((selectedRecord.successfulChains / selectedRecord.totalChains) * 100)}%
              </div>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div className="bg-blue-50 rounded-lg p-4 text-center">
              <div className="text-2xl font-bold text-blue-600">{selectedRecord.totalChains}</div>
              <div className="text-sm text-blue-700">Total Networks</div>
            </div>
            <div className="bg-green-50 rounded-lg p-4 text-center">
              <div className="text-2xl font-bold text-green-600">{selectedRecord.successfulChains}</div>
              <div className="text-sm text-green-700">Successful</div>
            </div>
            <div className="bg-red-50 rounded-lg p-4 text-center">
              <div className="text-2xl font-bold text-red-600">{selectedRecord.failedChains}</div>
              <div className="text-sm text-red-700">Failed</div>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-4">Deployment Details</h2>
          
          <div className="overflow-x-auto">
            <table className="w-full table-auto">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Status</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Network</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Contract Address</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Transaction Hash</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Gas Used</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Actions</th>
                </tr>
              </thead>
              <tbody>
                {selectedRecord.deployments.map((deployment, index) => (
                  <tr key={index} className="border-b border-gray-100">
                    <td className="py-3 px-4">
                      <div className="flex items-center space-x-2">
                        {getStatusIcon(deployment.status)}
                        <span className="capitalize">{deployment.status}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <div>
                        <div className="font-medium text-gray-900">{deployment.chainName}</div>
                        <div className="text-sm text-gray-500">ID: {deployment.chainId}</div>
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      {deployment.contractAddress ? (
                        <code className="text-sm bg-gray-100 px-2 py-1 rounded font-mono">
                          {deployment.contractAddress}
                        </code>
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="py-3 px-4">
                      {deployment.transactionHash ? (
                        <code className="text-sm bg-gray-100 px-2 py-1 rounded font-mono">
                          {deployment.transactionHash.slice(0, 10)}...
                        </code>
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-sm text-gray-600">
                      {deployment.gasUsed || '-'}
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center space-x-2">
                        {deployment.contractAddress && (
                          <button
                            onClick={() => copyToClipboard(deployment.contractAddress!)}
                            className="p-1 text-gray-400 hover:text-gray-600"
                            title="Copy contract address"
                          >
                            <Copy className="w-4 h-4" />
                          </button>
                        )}
                        {deployment.transactionHash && (
                          <button
                            onClick={() => copyToClipboard(deployment.transactionHash!)}
                            className="p-1 text-gray-400 hover:text-gray-600"
                            title="Copy transaction hash"
                          >
                            <ExternalLink className="w-4 h-4" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {selectedRecord.deployments.some(d => d.status === 'failed') && (
            <div className="mt-6">
              <h3 className="text-lg font-semibold text-red-900 mb-3">Failed Deployments</h3>
              <div className="space-y-2">
                {selectedRecord.deployments
                  .filter(d => d.status === 'failed')
                  .map((deployment, index) => (
                    <div key={index} className="p-3 bg-red-50 border border-red-200 rounded-lg">
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-red-900">{deployment.chainName}</span>
                        <span className="text-sm text-red-700">{deployment.error}</span>
                      </div>
                    </div>
                  ))}
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 flex items-center space-x-3">
            <History className="w-8 h-8" />
            <span>Deployment History</span>
          </h1>
          <p className="text-gray-600 mt-2">
            Track and manage your smart contract deployments across multiple networks
          </p>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="card p-6 mb-6">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between space-y-4 md:space-y-0 md:space-x-4">
          <div className="flex-1 max-w-md">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Search by contract name..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="input-field pl-10"
              />
            </div>
          </div>

          <div className="flex items-center space-x-2">
            <Filter className="w-5 h-5 text-gray-400" />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as any)}
              className="input-field w-auto"
            >
              <option value="all">All Deployments</option>
              <option value="success">Successful Only</option>
              <option value="failed">With Failures</option>
            </select>
          </div>
        </div>
      </div>

      {/* History List */}
      {filteredHistory.length === 0 ? (
        <div className="card p-12 text-center">
          <History className="w-16 h-16 text-gray-300 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-gray-900 mb-2">No deployment history</h3>
          <p className="text-gray-600">Your contract deployments will appear here</p>
        </div>
      ) : (
        <div className="space-y-4">
          {filteredHistory.map((record) => (
            <div
              key={record.id}
              className="card p-6 cursor-pointer hover:shadow-lg transition-shadow"
              onClick={() => setSelectedRecord(record)}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center space-x-3 mb-2">
                    <h3 className="text-xl font-semibold text-gray-900">{record.contractName}</h3>
                    <span className="text-sm text-gray-500">
                      {record.timestamp.toLocaleDateString()}
                    </span>
                  </div>
                  
                  <div className="flex items-center space-x-6 text-sm text-gray-600">
                    <div className="flex items-center space-x-1">
                      <span>Total Networks:</span>
                      <span className="font-medium">{record.totalChains}</span>
                    </div>
                    <div className="flex items-center space-x-1">
                      <CheckCircle className="w-4 h-4 text-green-500" />
                      <span>{record.successfulChains} successful</span>
                    </div>
                    {record.failedChains > 0 && (
                      <div className="flex items-center space-x-1">
                        <XCircle className="w-4 h-4 text-red-500" />
                        <span>{record.failedChains} failed</span>
                      </div>
                    )}
                  </div>
                </div>

                <div className="text-right">
                  <div className="text-2xl font-bold text-green-600">
                    {Math.round((record.successfulChains / record.totalChains) * 100)}%
                  </div>
                  <div className="text-sm text-gray-500">Success Rate</div>
                </div>
              </div>

              <div className="mt-4 pt-4 border-t border-gray-200">
                <div className="flex items-center justify-between">
                  <div className="text-sm text-gray-500">
                    Click to view detailed deployment information
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      exportDeploymentData(record);
                    }}
                    className="btn-secondary text-sm flex items-center space-x-1"
                  >
                    <Download className="w-4 h-4" />
                    <span>Export</span>
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};