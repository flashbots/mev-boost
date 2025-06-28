import React, { useState, useEffect } from 'react';
import { web3Service, WalletConnection as WalletConnectionType } from '../services/web3Service';
import { realTimeEarningService } from '../services/realTimeEarningService';
import { 
  Wallet, 
  Power, 
  Copy, 
  ExternalLink, 
  AlertCircle,
  CheckCircle,
  RefreshCw
} from 'lucide-react';

interface WalletConnectionProps {
  onConnectionChange: (connected: boolean) => void;
}

export const WalletConnection: React.FC<WalletConnectionProps> = ({ onConnectionChange }) => {
  const [connection, setConnection] = useState<WalletConnectionType | null>(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [balance, setBalance] = useState<string>('0');

  useEffect(() => {
    // Sayfa yüklendiğinde mevcut bağlantıyı kontrol et
    checkExistingConnection();
  }, []);

  const checkExistingConnection = async () => {
    if (window.ethereum && window.ethereum.selectedAddress) {
      try {
        const conn = await web3Service.connectWallet();
        setConnection(conn);
        setBalance(conn.balance);
        onConnectionChange(true);
        
        // Real-time tracking'i başlat
        await realTimeEarningService.startRealTimeTracking();
        
        // Bakiye takibini başlat
        web3Service.startBalanceTracking(setBalance);
      } catch (error) {
        console.error('Failed to restore connection:', error);
      }
    }
  };

  const connectWallet = async () => {
    setIsConnecting(true);
    setError(null);

    try {
      const conn = await web3Service.connectWallet();
      setConnection(conn);
      setBalance(conn.balance);
      onConnectionChange(true);

      // Real-time tracking'i başlat
      await realTimeEarningService.startRealTimeTracking();
      
      // Bakiye takibini başlat
      web3Service.startBalanceTracking(setBalance);

    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to connect wallet');
    } finally {
      setIsConnecting(false);
    }
  };

  const disconnectWallet = async () => {
    await web3Service.disconnectWallet();
    realTimeEarningService.stopRealTimeTracking();
    setConnection(null);
    setBalance('0');
    setError(null);
    onConnectionChange(false);
  };

  const copyAddress = () => {
    if (connection) {
      navigator.clipboard.writeText(connection.address);
    }
  };

  const switchToMainnet = async () => {
    try {
      await web3Service.switchNetwork(1);
    } catch (error) {
      setError('Failed to switch network');
    }
  };

  const getNetworkName = (chainId: number): string => {
    const networks: { [key: number]: string } = {
      1: 'Ethereum Mainnet',
      137: 'Polygon',
      56: 'BSC',
      43114: 'Avalanche',
      250: 'Fantom',
      42161: 'Arbitrum',
      10: 'Optimism',
      5: 'Goerli Testnet',
      80001: 'Mumbai Testnet'
    };
    return networks[chainId] || `Chain ${chainId}`;
  };

  const getNetworkColor = (chainId: number): string => {
    if (chainId === 1) return 'text-green-600 bg-green-100';
    if ([5, 80001, 97].includes(chainId)) return 'text-yellow-600 bg-yellow-100';
    return 'text-blue-600 bg-blue-100';
  };

  if (!connection) {
    return (
      <div className="card p-6">
        <div className="text-center">
          <div className="w-16 h-16 bg-gradient-to-br from-purple-500 to-blue-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <Wallet className="w-8 h-8 text-white" />
          </div>
          
          <h2 className="text-xl font-semibold text-gray-900 mb-2">Connect Your Wallet</h2>
          <p className="text-gray-600 mb-6">
            Connect your wallet to start earning ETH from multi-chain operations
          </p>

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center space-x-2">
              <AlertCircle className="w-5 h-5 text-red-500" />
              <span className="text-red-700 text-sm">{error}</span>
            </div>
          )}

          <button
            onClick={connectWallet}
            disabled={isConnecting}
            className="btn-primary flex items-center space-x-2 mx-auto"
          >
            {isConnecting ? (
              <RefreshCw className="w-5 h-5 animate-spin" />
            ) : (
              <Wallet className="w-5 h-5" />
            )}
            <span>{isConnecting ? 'Connecting...' : 'Connect MetaMask'}</span>
          </button>

          <div className="mt-6 text-sm text-gray-500">
            <p>Supported wallets:</p>
            <div className="flex items-center justify-center space-x-4 mt-2">
              <span className="px-3 py-1 bg-orange-100 text-orange-700 rounded-full">MetaMask</span>
              <span className="px-3 py-1 bg-blue-100 text-blue-700 rounded-full">WalletConnect</span>
              <span className="px-3 py-1 bg-purple-100 text-purple-700 rounded-full">Coinbase</span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="card p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900 flex items-center space-x-2">
          <CheckCircle className="w-5 h-5 text-green-500" />
          <span>Wallet Connected</span>
        </h2>
        
        <button
          onClick={disconnectWallet}
          className="btn-secondary text-sm flex items-center space-x-1"
        >
          <Power className="w-4 h-4" />
          <span>Disconnect</span>
        </button>
      </div>

      <div className="space-y-4">
        {/* Address */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Address</label>
          <div className="flex items-center space-x-2">
            <code className="flex-1 text-sm bg-gray-100 px-3 py-2 rounded font-mono">
              {connection.address}
            </code>
            <button
              onClick={copyAddress}
              className="p-2 text-gray-400 hover:text-gray-600"
              title="Copy address"
            >
              <Copy className="w-4 h-4" />
            </button>
            <a
              href={`https://etherscan.io/address/${connection.address}`}
              target="_blank"
              rel="noopener noreferrer"
              className="p-2 text-gray-400 hover:text-gray-600"
              title="View on Etherscan"
            >
              <ExternalLink className="w-4 h-4" />
            </a>
          </div>
        </div>

        {/* Network */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Network</label>
          <div className="flex items-center justify-between">
            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getNetworkColor(connection.chainId)}`}>
              {getNetworkName(connection.chainId)}
            </span>
            
            {connection.chainId !== 1 && (
              <button
                onClick={switchToMainnet}
                className="btn-secondary text-sm"
              >
                Switch to Mainnet
              </button>
            )}
          </div>
        </div>

        {/* Balance */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Balance</label>
          <div className="text-lg font-semibold text-gray-900">
            {parseFloat(balance).toFixed(4)} ETH
          </div>
          <div className="text-sm text-gray-500">
            ≈ ${(parseFloat(balance) * 3500).toFixed(2)} USD
          </div>
        </div>

        {/* Status */}
        <div className="pt-4 border-t border-gray-200">
          <div className="flex items-center space-x-2">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            <span className="text-sm text-green-600 font-medium">Real-time tracking active</span>
          </div>
          <p className="text-xs text-gray-500 mt-1">
            Monitoring transactions for automatic commission calculation
          </p>
        </div>
      </div>
    </div>
  );
};