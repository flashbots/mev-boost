import React, { useState, useEffect } from 'react';
import { realTimeEarningService, LiveStats, RealTimeEarning } from '../services/realTimeEarningService';
import { web3Service } from '../services/web3Service';
import { 
  DollarSign, 
  TrendingUp, 
  Clock, 
  Zap, 
  Activity,
  RefreshCw,
  Eye,
  EyeOff,
  AlertCircle,
  CheckCircle,
  ExternalLink
} from 'lucide-react';

export const RealTimeEarningsPanel: React.FC = () => {
  const [stats, setStats] = useState<LiveStats | null>(null);
  const [recentEarnings, setRecentEarnings] = useState<RealTimeEarning[]>([]);
  const [pendingTransactions, setPendingTransactions] = useState<RealTimeEarning[]>([]);
  const [showBalances, setShowBalances] = useState(true);
  const [isConnected, setIsConnected] = useState(false);
  const [ethPrice, setEthPrice] = useState(3500);

  useEffect(() => {
    // Wallet bağlantısını kontrol et
    const connection = web3Service.getConnection();
    setIsConnected(!!connection);

    if (connection) {
      // Stats update callback'ini kaydet
      realTimeEarningService.onStatsUpdate(handleStatsUpdate);
      
      // İlk verileri yükle
      loadInitialData();
      
      // ETH fiyatını güncelle
      updateETHPrice();
    }

    return () => {
      // Cleanup
    };
  }, []);

  const handleStatsUpdate = (newStats: LiveStats) => {
    setStats(newStats);
    setRecentEarnings(realTimeEarningService.getEarningsHistory(10));
    setPendingTransactions(realTimeEarningService.getPendingTransactions());
  };

  const loadInitialData = () => {
    const initialStats = realTimeEarningService.getLiveStats();
    setStats(initialStats);
    setRecentEarnings(realTimeEarningService.getEarningsHistory(10));
    setPendingTransactions(realTimeEarningService.getPendingTransactions());
  };

  const updateETHPrice = async () => {
    try {
      const price = await web3Service.fetchRealTimeETHPrice();
      setEthPrice(price);
    } catch (error) {
      console.error('Failed to update ETH price:', error);
    }
  };

  const getOperationIcon = (operation: string) => {
    switch (operation) {
      case 'deployment':
        return <Zap className="w-4 h-4 text-blue-500" />;
      case 'verification':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      default:
        return <Activity className="w-4 h-4 text-purple-500" />;
    }
  };

  const getOperationColor = (operation: string) => {
    switch (operation) {
      case 'deployment':
        return 'bg-blue-100 text-blue-800';
      case 'verification':
        return 'bg-green-100 text-green-800';
      default:
        return 'bg-purple-100 text-purple-800';
    }
  };

  if (!isConnected) {
    return (
      <div className="card p-8 text-center">
        <AlertCircle className="w-16 h-16 text-gray-300 mx-auto mb-4" />
        <h3 className="text-lg font-semibold text-gray-900 mb-2">Wallet Not Connected</h3>
        <p className="text-gray-600">
          Please connect your wallet to view real-time earnings
        </p>
      </div>
    );
  }

  if (!stats) {
    return (
      <div className="flex items-center justify-center py-8">
        <RefreshCw className="w-8 h-8 animate-spin text-primary-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 flex items-center space-x-3">
            <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-emerald-600 rounded-full flex items-center justify-center">
              <Activity className="w-6 h-6 text-white" />
            </div>
            <span>Real-Time Earnings</span>
          </h1>
          <p className="text-gray-600 mt-2">
            Live tracking of your blockchain earnings • Last updated: {stats.lastUpdated.toLocaleTimeString()}
          </p>
        </div>

        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowBalances(!showBalances)}
            className="btn-secondary flex items-center space-x-2"
          >
            {showBalances ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            <span>{showBalances ? 'Hide' : 'Show'}</span>
          </button>

          <div className="flex items-center space-x-2 text-sm text-gray-600">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            <span>Live</span>
          </div>
        </div>
      </div>

      {/* Live Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="card p-6 bg-gradient-to-br from-green-50 to-emerald-50 border-green-200">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center">
              <DollarSign className="w-6 h-6 text-green-600" />
            </div>
            <TrendingUp className="w-5 h-5 text-green-500" />
          </div>
          <div>
            <p className="text-sm text-green-700 mb-1">Total Earnings</p>
            <p className="text-2xl font-bold text-green-900">
              {showBalances ? `${stats.totalEarningsETH} ETH` : '••••••'}
            </p>
            <p className="text-sm text-green-600">
              {showBalances ? `$${(parseFloat(stats.totalEarningsETH) * ethPrice).toFixed(2)}` : '••••••'}
            </p>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
              <Clock className="w-6 h-6 text-blue-600" />
            </div>
          </div>
          <div>
            <p className="text-sm text-gray-600 mb-1">Today's Earnings</p>
            <p className="text-2xl font-bold text-gray-900">
              {showBalances ? `${stats.todayEarningsETH} ETH` : '••••••'}
            </p>
            <p className="text-sm text-gray-500">
              {showBalances ? `$${(parseFloat(stats.todayEarningsETH) * ethPrice).toFixed(2)}` : '••••••'}
            </p>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-purple-100 rounded-full flex items-center justify-center">
              <Activity className="w-6 h-6 text-purple-600" />
            </div>
          </div>
          <div>
            <p className="text-sm text-gray-600 mb-1">Total Transactions</p>
            <p className="text-2xl font-bold text-gray-900">{stats.totalTransactions}</p>
            <p className="text-sm text-gray-500">
              Avg Gas: {stats.averageGasPrice} Gwei
            </p>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-orange-100 rounded-full flex items-center justify-center">
              <RefreshCw className="w-6 h-6 text-orange-600" />
            </div>
          </div>
          <div>
            <p className="text-sm text-gray-600 mb-1">Pending</p>
            <p className="text-2xl font-bold text-gray-900">{stats.pendingTransactions}</p>
            <p className="text-sm text-gray-500">
              {showBalances ? `${stats.pendingEarningsETH} ETH` : '••••••'}
            </p>
          </div>
        </div>
      </div>

      {/* ETH Price */}
      <div className="card p-4 bg-gradient-to-r from-blue-50 to-indigo-50 border-blue-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
              <TrendingUp className="w-4 h-4 text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-blue-700">Current ETH Price</p>
              <p className="text-lg font-bold text-blue-900">${ethPrice.toLocaleString()}</p>
            </div>
          </div>
          <button
            onClick={updateETHPrice}
            className="btn-secondary text-sm"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Pending Transactions */}
      {pendingTransactions.length > 0 && (
        <div className="card p-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-4 flex items-center space-x-2">
            <RefreshCw className="w-5 h-5 text-orange-500 animate-spin" />
            <span>Pending Transactions</span>
          </h2>

          <div className="space-y-3">
            {pendingTransactions.map((tx) => (
              <div key={tx.id} className="flex items-center justify-between p-3 bg-orange-50 border border-orange-200 rounded-lg">
                <div className="flex items-center space-x-3">
                  {getOperationIcon(tx.operation)}
                  <div>
                    <p className="font-medium text-gray-900">{tx.chainName}</p>
                    <p className="text-sm text-gray-500 capitalize">{tx.operation}</p>
                  </div>
                </div>
                <div className="text-right">
                  <code className="text-xs bg-gray-100 px-2 py-1 rounded font-mono">
                    {tx.txHash.slice(0, 10)}...
                  </code>
                  <p className="text-xs text-orange-600 mt-1">Confirming...</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Recent Earnings */}
      <div className="card p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-6 flex items-center space-x-2">
          <Activity className="w-5 h-5" />
          <span>Recent Earnings</span>
        </h2>

        {recentEarnings.length === 0 ? (
          <div className="text-center py-8">
            <Activity className="w-12 h-12 text-gray-300 mx-auto mb-4" />
            <p className="text-gray-600">No earnings yet. Start deploying contracts to earn commissions!</p>
          </div>
        ) : (
          <div className="space-y-3">
            {recentEarnings.map((earning) => (
              <div key={earning.id} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
                <div className="flex items-center space-x-4">
                  {getOperationIcon(earning.operation)}
                  <div>
                    <div className="flex items-center space-x-2">
                      <p className="font-medium text-gray-900">{earning.chainName}</p>
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${getOperationColor(earning.operation)}`}>
                        {earning.operation}
                      </span>
                    </div>
                    <p className="text-sm text-gray-500">
                      Block #{earning.blockNumber} • {earning.timestamp.toLocaleTimeString()}
                    </p>
                  </div>
                </div>
                
                <div className="text-right">
                  <p className="font-semibold text-green-600">
                    {showBalances ? `+${earning.commissionETH} ETH` : '+••••••'}
                  </p>
                  <div className="flex items-center space-x-2 mt-1">
                    <code className="text-xs bg-gray-200 px-2 py-1 rounded font-mono">
                      {earning.txHash.slice(0, 8)}...
                    </code>
                    <a
                      href={`https://etherscan.io/tx/${earning.txHash}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-gray-400 hover:text-gray-600"
                    >
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Weekly/Monthly Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Weekly Performance</h3>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-gray-600">Total Earnings:</span>
              <span className="font-semibold">
                {showBalances ? `${stats.weeklyEarningsETH} ETH` : '••••••'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">USD Value:</span>
              <span className="font-semibold">
                {showBalances ? `$${(parseFloat(stats.weeklyEarningsETH) * ethPrice).toFixed(2)}` : '••••••'}
              </span>
            </div>
          </div>
        </div>

        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Monthly Performance</h3>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-gray-600">Total Earnings:</span>
              <span className="font-semibold">
                {showBalances ? `${stats.monthlyEarningsETH} ETH` : '••••••'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">USD Value:</span>
              <span className="font-semibold">
                {showBalances ? `$${(parseFloat(stats.monthlyEarningsETH) * ethPrice).toFixed(2)}` : '••••••'}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};