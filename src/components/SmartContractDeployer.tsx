import React, { useState, useEffect } from 'react';
import { ChainData } from '../types/chain';
import { chainService } from '../services/chainService';
import { 
  Code, 
  Upload, 
  Play, 
  CheckCircle, 
  XCircle, 
  Clock, 
  AlertTriangle,
  Copy,
  ExternalLink,
  Settings,
  Zap
} from 'lucide-react';

interface DeploymentStatus {
  chainId: number;
  chainName: string;
  status: 'pending' | 'deploying' | 'success' | 'failed';
  txHash?: string;
  contractAddress?: string;
  error?: string;
  gasUsed?: string;
  deploymentTime?: number;
}

interface SmartContractDeployerProps {
  chains: ChainData[];
}

export const SmartContractDeployer: React.FC<SmartContractDeployerProps> = ({ chains }) => {
  const [contractCode, setContractCode] = useState(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract MyToken {
    string public name = "My Token";
    string public symbol = "MTK";
    uint8 public decimals = 18;
    uint256 public totalSupply = 1000000 * 10**decimals;
    
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    
    constructor() {
        balanceOf[msg.sender] = totalSupply;
        emit Transfer(address(0), msg.sender, totalSupply);
    }
    
    function transfer(address to, uint256 value) public returns (bool) {
        require(balanceOf[msg.sender] >= value, "Insufficient balance");
        balanceOf[msg.sender] -= value;
        balanceOf[to] += value;
        emit Transfer(msg.sender, to, value);
        return true;
    }
    
    function approve(address spender, uint256 value) public returns (bool) {
        allowance[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }
    
    function transferFrom(address from, address to, uint256 value) public returns (bool) {
        require(balanceOf[from] >= value, "Insufficient balance");
        require(allowance[from][msg.sender] >= value, "Insufficient allowance");
        
        balanceOf[from] -= value;
        balanceOf[to] += value;
        allowance[from][msg.sender] -= value;
        
        emit Transfer(from, to, value);
        return true;
    }
}`);

  const [selectedChains, setSelectedChains] = useState<number[]>([]);
  const [deploymentStatuses, setDeploymentStatuses] = useState<DeploymentStatus[]>([]);
  const [isDeploying, setIsDeploying] = useState(false);
  const [privateKey, setPrivateKey] = useState('');
  const [gasLimit, setGasLimit] = useState('3000000');
  const [gasPrice, setGasPrice] = useState('20'); // Gwei
  const [constructorArgs, setConstructorArgs] = useState('');

  // Deployment konfigürasyonu
  const [deploymentConfig, setDeploymentConfig] = useState({
    batchSize: 5, // Aynı anda kaç ağa deploy edilecek
    retryAttempts: 3,
    delayBetweenDeployments: 2000, // ms
    onlyTestnets: false,
    onlyMainnets: false
  });

  const availableChains = chains.filter(chain => {
    if (deploymentConfig.onlyTestnets) {
      return chain.name.toLowerCase().includes('test') || 
             chain.chain.toLowerCase().includes('test');
    }
    if (deploymentConfig.onlyMainnets) {
      return !(chain.name.toLowerCase().includes('test') || 
               chain.chain.toLowerCase().includes('test'));
    }
    return chain.rpc && chain.rpc.length > 0;
  });

  const handleChainSelection = (chainId: number) => {
    setSelectedChains(prev => 
      prev.includes(chainId) 
        ? prev.filter(id => id !== chainId)
        : [...prev, chainId]
    );
  };

  const selectAllChains = () => {
    setSelectedChains(availableChains.map(chain => chain.chainId));
  };

  const clearSelection = () => {
    setSelectedChains([]);
  };

  const compileContract = async (sourceCode: string) => {
    // Simulated compilation - gerçek uygulamada Solidity compiler kullanılır
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          bytecode: '0x608060405234801561001057600080fd5b50...',
          abi: [
            {
              "inputs": [],
              "name": "name",
              "outputs": [{"internalType": "string", "name": "", "type": "string"}],
              "stateMutability": "view",
              "type": "function"
            }
          ]
        });
      }, 1000);
    });
  };

  const deployToChain = async (chain: ChainData, compiledContract: any): Promise<DeploymentStatus> => {
    const startTime = Date.now();
    
    try {
      // RPC endpoint seç
      const workingRpc = await findWorkingRpc(chain);
      if (!workingRpc) {
        throw new Error('No working RPC endpoint found');
      }

      // Simulated deployment - gerçek uygulamada Web3/Ethers.js kullanılır
      await new Promise(resolve => setTimeout(resolve, 3000 + Math.random() * 2000));

      // Random success/failure simulation
      const success = Math.random() > 0.2; // %80 başarı oranı

      if (success) {
        const mockTxHash = '0x' + Array.from({length: 64}, () => Math.floor(Math.random() * 16).toString(16)).join('');
        const mockContractAddress = '0x' + Array.from({length: 40}, () => Math.floor(Math.random() * 16).toString(16)).join('');
        
        return {
          chainId: chain.chainId,
          chainName: chain.name,
          status: 'success',
          txHash: mockTxHash,
          contractAddress: mockContractAddress,
          gasUsed: (Math.floor(Math.random() * 500000) + 100000).toString(),
          deploymentTime: Date.now() - startTime
        };
      } else {
        throw new Error('Transaction failed: insufficient gas or network error');
      }
    } catch (error) {
      return {
        chainId: chain.chainId,
        chainName: chain.name,
        status: 'failed',
        error: error instanceof Error ? error.message : 'Unknown error',
        deploymentTime: Date.now() - startTime
      };
    }
  };

  const findWorkingRpc = async (chain: ChainData): Promise<string | null> => {
    for (const rpc of chain.rpc) {
      try {
        const result = await chainService.testRpcEndpoint(rpc);
        if (result.status === 'online') {
          return rpc;
        }
      } catch (error) {
        continue;
      }
    }
    return null;
  };

  const startDeployment = async () => {
    if (!contractCode.trim() || selectedChains.length === 0) {
      alert('Please provide contract code and select at least one chain');
      return;
    }

    setIsDeploying(true);
    setDeploymentStatuses([]);

    try {
      // Contract'ı compile et
      console.log('Compiling contract...');
      const compiledContract = await compileContract(contractCode);

      // Seçilen chain'leri al
      const chainsToDeployTo = chains.filter(chain => selectedChains.includes(chain.chainId));

      // Initial statuses
      const initialStatuses: DeploymentStatus[] = chainsToDeployTo.map(chain => ({
        chainId: chain.chainId,
        chainName: chain.name,
        status: 'pending'
      }));
      setDeploymentStatuses(initialStatuses);

      // Batch deployment
      const batches = [];
      for (let i = 0; i < chainsToDeployTo.length; i += deploymentConfig.batchSize) {
        batches.push(chainsToDeployTo.slice(i, i + deploymentConfig.batchSize));
      }

      for (const batch of batches) {
        // Batch içindeki tüm deployment'ları paralel başlat
        const deploymentPromises = batch.map(async (chain) => {
          // Status'u deploying olarak güncelle
          setDeploymentStatuses(prev => 
            prev.map(status => 
              status.chainId === chain.chainId 
                ? { ...status, status: 'deploying' }
                : status
            )
          );

          // Deploy et
          const result = await deployToChain(chain, compiledContract);
          
          // Status'u güncelle
          setDeploymentStatuses(prev => 
            prev.map(status => 
              status.chainId === chain.chainId ? result : status
            )
          );

          return result;
        });

        // Batch'i bekle
        await Promise.all(deploymentPromises);

        // Batch'ler arası delay
        if (batches.indexOf(batch) < batches.length - 1) {
          await new Promise(resolve => setTimeout(resolve, deploymentConfig.delayBetweenDeployments));
        }
      }

    } catch (error) {
      console.error('Deployment failed:', error);
      alert('Deployment failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setIsDeploying(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="w-5 h-5 text-green-500" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-500" />;
      case 'deploying':
        return <Clock className="w-5 h-5 text-blue-500 animate-spin" />;
      default:
        return <Clock className="w-5 h-5 text-gray-400" />;
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const successfulDeployments = deploymentStatuses.filter(s => s.status === 'success');
  const failedDeployments = deploymentStatuses.filter(s => s.status === 'failed');

  return (
    <div className="max-w-7xl mx-auto space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="flex items-center justify-center space-x-3 mb-4">
          <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-blue-600 rounded-full flex items-center justify-center">
            <Code className="w-6 h-6 text-white" />
          </div>
          <h1 className="text-3xl font-bold text-gray-900">Multi-Chain Smart Contract Deployer</h1>
        </div>
        <p className="text-lg text-gray-600">
          Deploy your smart contracts to multiple blockchain networks simultaneously
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Contract Code Editor */}
        <div className="lg:col-span-2 space-y-6">
          <div className="card p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4 flex items-center space-x-2">
              <Code className="w-5 h-5" />
              <span>Smart Contract Code</span>
            </h2>
            
            <textarea
              value={contractCode}
              onChange={(e) => setContractCode(e.target.value)}
              className="w-full h-96 font-mono text-sm border border-gray-300 rounded-lg p-4 focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              placeholder="Enter your Solidity contract code here..."
            />

            <div className="mt-4 flex items-center justify-between">
              <div className="text-sm text-gray-500">
                Lines: {contractCode.split('\n').length} | Characters: {contractCode.length}
              </div>
              <button
                onClick={() => setContractCode('')}
                className="btn-secondary text-sm"
              >
                Clear
              </button>
            </div>
          </div>

          {/* Deployment Configuration */}
          <div className="card p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4 flex items-center space-x-2">
              <Settings className="w-5 h-5" />
              <span>Deployment Configuration</span>
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Gas Limit
                </label>
                <input
                  type="text"
                  value={gasLimit}
                  onChange={(e) => setGasLimit(e.target.value)}
                  className="input-field"
                  placeholder="3000000"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Gas Price (Gwei)
                </label>
                <input
                  type="text"
                  value={gasPrice}
                  onChange={(e) => setGasPrice(e.target.value)}
                  className="input-field"
                  placeholder="20"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Batch Size
                </label>
                <input
                  type="number"
                  value={deploymentConfig.batchSize}
                  onChange={(e) => setDeploymentConfig(prev => ({
                    ...prev,
                    batchSize: parseInt(e.target.value) || 5
                  }))}
                  className="input-field"
                  min="1"
                  max="20"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Retry Attempts
                </label>
                <input
                  type="number"
                  value={deploymentConfig.retryAttempts}
                  onChange={(e) => setDeploymentConfig(prev => ({
                    ...prev,
                    retryAttempts: parseInt(e.target.value) || 3
                  }))}
                  className="input-field"
                  min="0"
                  max="10"
                />
              </div>
            </div>

            <div className="mt-4 space-y-3">
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={deploymentConfig.onlyTestnets}
                  onChange={(e) => setDeploymentConfig(prev => ({
                    ...prev,
                    onlyTestnets: e.target.checked,
                    onlyMainnets: e.target.checked ? false : prev.onlyMainnets
                  }))}
                  className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span className="text-sm text-gray-700">Deploy only to testnets</span>
              </label>

              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={deploymentConfig.onlyMainnets}
                  onChange={(e) => setDeploymentConfig(prev => ({
                    ...prev,
                    onlyMainnets: e.target.checked,
                    onlyTestnets: e.target.checked ? false : prev.onlyTestnets
                  }))}
                  className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span className="text-sm text-gray-700">Deploy only to mainnets</span>
              </label>
            </div>
          </div>
        </div>

        {/* Chain Selection */}
        <div className="space-y-6">
          <div className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900 flex items-center space-x-2">
                <Zap className="w-5 h-5" />
                <span>Select Networks</span>
              </h2>
              <div className="text-sm text-gray-500">
                {selectedChains.length} / {availableChains.length}
              </div>
            </div>

            <div className="flex space-x-2 mb-4">
              <button
                onClick={selectAllChains}
                className="btn-secondary text-sm flex-1"
              >
                Select All
              </button>
              <button
                onClick={clearSelection}
                className="btn-secondary text-sm flex-1"
              >
                Clear
              </button>
            </div>

            <div className="max-h-96 overflow-y-auto space-y-2">
              {availableChains.map((chain) => (
                <label
                  key={chain.chainId}
                  className="flex items-center space-x-3 p-3 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={selectedChains.includes(chain.chainId)}
                    onChange={() => handleChainSelection(chain.chainId)}
                    className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                  />
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-gray-900 truncate">
                      {chain.name}
                    </div>
                    <div className="text-sm text-gray-500">
                      ID: {chain.chainId} | RPCs: {chain.rpc?.length || 0}
                    </div>
                  </div>
                </label>
              ))}
            </div>
          </div>

          {/* Deploy Button */}
          <button
            onClick={startDeployment}
            disabled={isDeploying || selectedChains.length === 0 || !contractCode.trim()}
            className="w-full btn-primary py-4 text-lg font-semibold flex items-center justify-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isDeploying ? (
              <>
                <Clock className="w-5 h-5 animate-spin" />
                <span>Deploying...</span>
              </>
            ) : (
              <>
                <Play className="w-5 h-5" />
                <span>Deploy to {selectedChains.length} Networks</span>
              </>
            )}
          </button>

          {/* Deployment Summary */}
          {deploymentStatuses.length > 0 && (
            <div className="card p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Deployment Summary</h3>
              
              <div className="grid grid-cols-3 gap-4 mb-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">{successfulDeployments.length}</div>
                  <div className="text-sm text-gray-500">Success</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">{failedDeployments.length}</div>
                  <div className="text-sm text-gray-500">Failed</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">
                    {deploymentStatuses.filter(s => s.status === 'deploying').length}
                  </div>
                  <div className="text-sm text-gray-500">Deploying</div>
                </div>
              </div>

              <div className="space-y-2 max-h-64 overflow-y-auto">
                {deploymentStatuses.map((status) => (
                  <div
                    key={status.chainId}
                    className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
                  >
                    <div className="flex items-center space-x-3">
                      {getStatusIcon(status.status)}
                      <div>
                        <div className="font-medium text-gray-900">{status.chainName}</div>
                        <div className="text-sm text-gray-500">ID: {status.chainId}</div>
                      </div>
                    </div>
                    
                    {status.status === 'success' && status.contractAddress && (
                      <button
                        onClick={() => copyToClipboard(status.contractAddress!)}
                        className="p-1 text-gray-400 hover:text-gray-600"
                        title="Copy contract address"
                      >
                        <Copy className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Deployment Results */}
      {successfulDeployments.length > 0 && (
        <div className="card p-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-6 flex items-center space-x-2">
            <CheckCircle className="w-5 h-5 text-green-500" />
            <span>Successful Deployments</span>
          </h2>

          <div className="overflow-x-auto">
            <table className="w-full table-auto">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Network</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Contract Address</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Transaction Hash</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Gas Used</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Time</th>
                  <th className="text-left py-3 px-4 font-medium text-gray-900">Actions</th>
                </tr>
              </thead>
              <tbody>
                {successfulDeployments.map((deployment) => (
                  <tr key={deployment.chainId} className="border-b border-gray-100">
                    <td className="py-3 px-4">
                      <div>
                        <div className="font-medium text-gray-900">{deployment.chainName}</div>
                        <div className="text-sm text-gray-500">ID: {deployment.chainId}</div>
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <code className="text-sm bg-gray-100 px-2 py-1 rounded font-mono">
                        {deployment.contractAddress}
                      </code>
                    </td>
                    <td className="py-3 px-4">
                      <code className="text-sm bg-gray-100 px-2 py-1 rounded font-mono">
                        {deployment.txHash?.slice(0, 10)}...
                      </code>
                    </td>
                    <td className="py-3 px-4 text-sm text-gray-600">
                      {deployment.gasUsed}
                    </td>
                    <td className="py-3 px-4 text-sm text-gray-600">
                      {deployment.deploymentTime}ms
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={() => copyToClipboard(deployment.contractAddress!)}
                          className="p-1 text-gray-400 hover:text-gray-600"
                          title="Copy address"
                        >
                          <Copy className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => copyToClipboard(deployment.txHash!)}
                          className="p-1 text-gray-400 hover:text-gray-600"
                          title="Copy tx hash"
                        >
                          <ExternalLink className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Failed Deployments */}
      {failedDeployments.length > 0 && (
        <div className="card p-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-6 flex items-center space-x-2">
            <XCircle className="w-5 h-5 text-red-500" />
            <span>Failed Deployments</span>
          </h2>

          <div className="space-y-4">
            {failedDeployments.map((deployment) => (
              <div key={deployment.chainId} className="p-4 bg-red-50 border border-red-200 rounded-lg">
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="font-medium text-red-900">{deployment.chainName}</h3>
                    <p className="text-sm text-red-700 mt-1">{deployment.error}</p>
                  </div>
                  <span className="text-sm text-red-600">ID: {deployment.chainId}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};