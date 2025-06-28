import React, { useState, useEffect } from 'react';
import { 
  DollarSign, 
  TrendingUp, 
  Wallet, 
  Download, 
  Settings,
  Eye,
  EyeOff,
  RefreshCw,
  ArrowUpRight,
  PieChart,
  BarChart3,
  Calendar
} from 'lucide-react';
import { ethereumEarningService, EarningStats, EarningRecord } from '../services/ethereumEarningService';

export const EarningsPanel: React.FC = () => {
  const [stats, setStats] = useState<EarningStats | null>(null);
  const [recentEarnings, setRecentEarnings] = useState<EarningRecord[]>([]);
  const [showBalances, setShowBalances] = useState(true);
  const [isWithdrawing, setIsWithdrawing] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [selectedTimeframe, setSelectedTimeframe] = useState<'today' | 'week' | 'month' | 'all'>('today');

  useEffect(() => {
    loadEarningsData();
    // ETH fiyatını güncelle
    ethereumEarningService.updateETHPrice();
    
    // Demo için mock data oluştur
    ethereumEarningService.generateMockEarnings();
  }, []);

  const loadEarningsData = () => {
    const earningStats = ethereumEarningService.getEarningStats();
    const earnings = ethereumEarningService.getEarningsHistory(20);
    
    setStats(earningStats);
    setRecentEarnings(earnings);
  };

  const handleWithdraw = async () => {
    setIsWithdrawing(true);
    
    try {
      const result = await ethereumEarningService.withdrawEarnings();
      
      if (result.success) {
        alert(`Successfully withdrew ${result.amount} ETH!\nTransaction: ${result.txHash}`);
        loadEarningsData(); // Refresh data
      } else {
        alert(`Withdrawal failed: ${result.error}`);
      }
    } catch (error) {
      alert('Withdrawal failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setIsWithdrawing(false);
    }
  };

  const getTimeframeEarnings = () => {
    if (!stats) return '0.000000';
    
    switch (selectedTimeframe) {
      case 'today':
        return stats.todayEarningsETH;
      case 'week':
        return stats.weeklyEarningsETH;
      case 'month':
        return stats.monthlyEarningsETH;
      case 'all':
        return stats.totalEarningsETH;
      default:
        return stats.todayEarningsETH;
    }
  };

  const getTimeframeLabel = () => {
    switch (selectedTimeframe) {
      case 'today':
        return 'Today';
      case 'week':
        return 'This Week';
      case 'month':
        return 'This Month';
      case 'all':
        return 'All Time';
      default:
        return 'Today';
    }
  };

  if (!stats) {
    return (
      <div className="flex items-center justify-center py-8">
        <RefreshCw className="w-8 h-8 animate-spin text-primary-600" />
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 flex items-center space-x-3">
            <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-emerald-600 rounded-full flex items-center justify-center">
              <DollarSign className="w-6 h-6 text-white" />
            </div>
            <span>Ethereum Earnings</span>
          </h1>
          <p className="text-gray-600 mt-2">
            Track your earnings from multi-chain operations and smart contract deployments
          </p>
        </div>

        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowBalances(!showBalances)}
            className="btn-secondary flex items-center space-x-2"
          >
            {showBalances ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            <span>{showBalances ? 'Hide' : 'Show'} Balances</span>
          </button>

          <button
            onClick={() => setShowSettings(!showSettings)}
            className="btn-secondary flex items-center space-x-2"
          >
            <Settings className="w-4 h-4" />
            <span>Settings</span>
          </button>

          <button
            onClick={handleWithdraw}
            disabled={isWithdrawing || parseFloat(stats.totalEarningsETH) === 0}
            className="btn-primary flex items-center space-x-2 disabled:opacity-50"
          >
            {isWithdrawing ? (
              <RefreshCw className="w-4 h-4 animate-spin" />
            ) : (
              <Download className="w-4 h-4" />
            )}
            <span>Withdraw</span>
          </button>
        </div>
      </div>

      {/* Main Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="card p-6 bg-gradient-to-br from-green-50 to-emerald-50 border-green-200">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center">
              <Wallet className="w-6 h-6 text-green-600" />
            </div>
            <TrendingUp className="w-5 h-5 text-green-500" />
          </div>
          <div>
            <p className="text-sm text-green-700 mb-1">Total Balance</p>
            <p className="text-2xl font-bold text-green-900">
              {showBalances ? `${stats.totalEarningsETH} ETH` : '••••••'}
            </p>
            <p className="text-sm text-green-600">
              {showBalances ? `$${stats.totalEarningsUSD}` : '••••••'}
            </p>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
              <Calendar className="w-6 h-6 text-blue-600" />
            </div>
            <select
              value={selectedTimeframe}
              onChange={(e) => setSelectedTimeframe(e.target.value as any)}
              className="text-sm border border-gray-300 rounded px-2 py-1"
            >
              <option value="today">Today</option>
              <option value="week">Week</option>
              <option value="month">Month</option>
              <option value="all">All Time</option>
            </select>
          </div>
          <div>
            <p className="text-sm text-gray-600 mb-1">{getTimeframeLabel()} Earnings</p>
            <p className="text-2xl font-bold text-gray-900">
              {showBalances ? `${getTimeframeEarnings()} ETH` : '••••••'}
            </p>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-purple-100 rounded-full flex items-center justify-center">
              <BarChart3 className="w-6 h-6 text-purple-600" />
            </div>
          </div>
          <div>
            <p className="text-sm text-gray-600 mb-1">Total Transactions</p>
            <p className="text-2xl font-bold text-gray-900">{stats.totalTransactions.toLocaleString()}</p>
            <p className="text-sm text-gray-500">
              Avg: {showBalances ? `${stats.averageCommission} ETH` : '••••••'}
            </p>
          </div>
        </div>

        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="w-12 h-12 bg-orange-100 rounded-full flex items-center justify-center">
              <PieChart className="w-6 h-6 text-orange-600" />
            </div>
          </div>
          <div>
            <p className="text-sm text-gray-600 mb-1">Commission Rate</p>
            <p className="text-2xl font-bold text-gray-900">2.5%</p>
            <p className="text-sm text-gray-500">Per transaction</p>
          </div>
        </div>
      </div>

      {/* Top Earning Chains */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card p-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-6 flex items-center space-x-2">
            <PieChart className="w-5 h-5" />
            <span>Top Earning Networks</span>
          </h2>

          <div className="space-y-4">
            {stats.topEarningChains.slice(0, 5).map((chain, index) => (
              <div key={chain.chainId} className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-primary-700 rounded-full flex items-center justify-center text-white text-sm font-bold">
                    {index + 1}
                  </div>
                  <div>
                    <p className="font-medium text-gray-900">{chain.chainName}</p>
                    <p className="text-sm text-gray-500">ID: {chain.chainId}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-gray-900">
                    {showBalances ? `${chain.earnings} ETH` : '••••••'}
                  </p>
                  <p className="text-sm text-gray-500">{chain.percentage.toFixed(1)}%</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Recent Earnings */}
        <div className="card p-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-6 flex items-center space-x-2">
            <ArrowUpRight className="w-5 h-5" />
            <span>Recent Earnings</span>
          </h2>

          <div className="space-y-3 max-h-80 overflow-y-auto">
            {recentEarnings.slice(0, 10).map((earning) => (
              <div key={earning.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-3">
                  <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                  <div>
                    <p className="font-medium text-gray-900">{earning.chainName}</p>
                    <p className="text-sm text-gray-500 capitalize">{earning.operation}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-green-600">
                    {showBalances ? `+${earning.commissionInETH} ETH` : '+••••••'}
                  </p>
                  <p className="text-xs text-gray-500">
                    {earning.timestamp.toLocaleDateString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Detailed Earnings Table */}
      <div className="card p-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-gray-900">Earnings History</h2>
          <button className="btn-secondary text-sm flex items-center space-x-2">
            <Download className="w-4 h-4" />
            <span>Export CSV</span>
          </button>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full table-auto">
            <thead>
              <tr className="border-b border-gray-200">
                <th className="text-left py-3 px-4 font-medium text-gray-900">Date</th>
                <th className="text-left py-3 px-4 font-medium text-gray-900">Network</th>
                <th className="text-left py-3 px-4 font-medium text-gray-900">Operation</th>
                <th className="text-left py-3 px-4 font-medium text-gray-900">Gas Used</th>
                <th className="text-left py-3 px-4 font-medium text-gray-900">Commission</th>
                <th className="text-left py-3 px-4 font-medium text-gray-900">ETH Earned</th>
                <th className="text-left py-3 px-4 font-medium text-gray-900">Transaction</th>
              </tr>
            </thead>
            <tbody>
              {recentEarnings.slice(0, 15).map((earning) => (
                <tr key={earning.id} className="border-b border-gray-100 hover:bg-gray-50">
                  <td className="py-3 px-4 text-sm text-gray-600">
                    {earning.timestamp.toLocaleDateString()}
                  </td>
                  <td className="py-3 px-4">
                    <div>
                      <p className="font-medium text-gray-900">{earning.chainName}</p>
                      <p className="text-xs text-gray-500">ID: {earning.chainId}</p>
                    </div>
                  </td>
                  <td className="py-3 px-4">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 capitalize">
                      {earning.operation}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-sm text-gray-600">
                    {parseInt(earning.gasUsed).toLocaleString()}
                  </td>
                  <td className="py-3 px-4 text-sm text-gray-600">
                    {showBalances ? earning.commission : '••••••'}
                  </td>
                  <td className="py-3 px-4">
                    <span className="font-semibold text-green-600">
                      {showBalances ? `${earning.commissionInETH} ETH` : '••••••'}
                    </span>
                  </td>
                  <td className="py-3 px-4">
                    <code className="text-xs bg-gray-100 px-2 py-1 rounded font-mono">
                      {earning.txHash.slice(0, 10)}...
                    </code>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Settings Modal */}
      {showSettings && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Earnings Settings</h3>
              <button
                onClick={() => setShowSettings(false)}
                className="text-gray-400 hover:text-gray-600"
              >
                ×
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Commission Rate (%)
                </label>
                <input
                  type="number"
                  step="0.1"
                  defaultValue="2.5"
                  className="input-field"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Auto Withdraw Threshold (ETH)
                </label>
                <input
                  type="number"
                  step="0.1"
                  defaultValue="1.0"
                  className="input-field"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Treasury Address
                </label>
                <input
                  type="text"
                  defaultValue="0x742d35Cc6634C0532925a3b8D4C9db96c4b4d8b6"
                  className="input-field font-mono text-sm"
                />
              </div>

              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  defaultChecked
                  className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span className="text-sm text-gray-700">Enable auto withdrawal</span>
              </label>
            </div>

            <div className="flex space-x-3 mt-6">
              <button
                onClick={() => setShowSettings(false)}
                className="btn-secondary flex-1"
              >
                Cancel
              </button>
              <button
                onClick={() => setShowSettings(false)}
                className="btn-primary flex-1"
              >
                Save Settings
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};