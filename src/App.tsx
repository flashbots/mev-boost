import React, { useState, useEffect, useMemo } from 'react';
import { ChainData } from './types/chain';
import { chainService } from './services/chainService';
import { ChainCard } from './components/ChainCard';
import { ChainDetail } from './components/ChainDetail';
import { SearchAndFilter } from './components/SearchAndFilter';
import { LoadingSpinner } from './components/LoadingSpinner';
import { ErrorMessage } from './components/ErrorMessage';
import { SmartContractDeployer } from './components/SmartContractDeployer';
import { DeploymentHistory } from './components/DeploymentHistory';
import { WalletConnection } from './components/WalletConnection';
import { RealTimeEarningsPanel } from './components/RealTimeEarningsPanel';
import { Link, Globe, Zap, Code, History, DollarSign, Wallet } from 'lucide-react';

type AppView = 'chains' | 'deployer' | 'history' | 'earnings';

function App() {
  const [chains, setChains] = useState<ChainData[]>([]);
  const [selectedChain, setSelectedChain] = useState<ChainData | null>(null);
  const [currentView, setCurrentView] = useState<AppView>('earnings');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState<'name' | 'chainId' | 'rpcCount'>('name');
  const [filters, setFilters] = useState<{
    hasTestnet?: boolean;
    hasMainnet?: boolean;
    hasRpc?: boolean;
    hasExplorer?: boolean;
  }>({});
  const [isWalletConnected, setIsWalletConnected] = useState(false);

  useEffect(() => {
    loadChains();
  }, []);

  const loadChains = async () => {
    try {
      setLoading(true);
      setError(null);
      const chainsData = await chainService.fetchChains();
      setChains(chainsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load chains');
    } finally {
      setLoading(false);
    }
  };

  const filteredAndSortedChains = useMemo(() => {
    let result = chainService.searchChains(chains, searchQuery);
    result = chainService.filterChains(result, filters);
    result = chainService.sortChains(result, sortBy);
    return result;
  }, [chains, searchQuery, filters, sortBy]);

  const stats = useMemo(() => {
    const totalRpcs = chains.reduce((sum, chain) => sum + (chain.rpc?.length || 0), 0);
    const chainsWithExplorers = chains.filter(chain => chain.explorers && chain.explorers.length > 0).length;
    const testnets = chains.filter(chain => 
      chain.name.toLowerCase().includes('test') || 
      chain.chain.toLowerCase().includes('test')
    ).length;

    return {
      totalChains: chains.length,
      totalRpcs,
      chainsWithExplorers,
      testnets,
      mainnets: chains.length - testnets
    };
  }, [chains]);

  const renderNavigation = () => (
    <nav className="bg-white border-b border-gray-200 sticky top-0 z-50">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center space-x-8">
            <div className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-gradient-to-br from-purple-500 to-blue-600 rounded-lg flex items-center justify-center">
                <Link className="w-4 h-4 text-white" />
              </div>
              <span className="text-xl font-bold text-gray-900">Multi-Chain RPC</span>
            </div>

            <div className="flex items-center space-x-1">
              <button
                onClick={() => setCurrentView('earnings')}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  currentView === 'earnings'
                    ? 'bg-primary-100 text-primary-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                }`}
              >
                <div className="flex items-center space-x-2">
                  <DollarSign className="w-4 h-4" />
                  <span>Real-Time Earnings</span>
                  {isWalletConnected && (
                    <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                  )}
                </div>
              </button>

              <button
                onClick={() => setCurrentView('deployer')}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  currentView === 'deployer'
                    ? 'bg-primary-100 text-primary-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                }`}
              >
                <div className="flex items-center space-x-2">
                  <Code className="w-4 h-4" />
                  <span>Smart Contract Deployer</span>
                </div>
              </button>

              <button
                onClick={() => {
                  setCurrentView('chains');
                  setSelectedChain(null);
                }}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  currentView === 'chains'
                    ? 'bg-primary-100 text-primary-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                }`}
              >
                <div className="flex items-center space-x-2">
                  <Globe className="w-4 h-4" />
                  <span>Networks</span>
                </div>
              </button>

              <button
                onClick={() => setCurrentView('history')}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  currentView === 'history'
                    ? 'bg-primary-100 text-primary-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                }`}
              >
                <div className="flex items-center space-x-2">
                  <History className="w-4 h-4" />
                  <span>History</span>
                </div>
              </button>
            </div>
          </div>

          <div className="flex items-center space-x-4">
            <div className="text-sm text-gray-500">
              {stats.totalChains} Networks • {stats.totalRpcs} RPCs
            </div>
            
            {!isWalletConnected && (
              <div className="flex items-center space-x-2 text-orange-600 bg-orange-50 px-3 py-1 rounded-full">
                <Wallet className="w-4 h-4" />
                <span className="text-sm font-medium">Connect Wallet</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </nav>
  );

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50">
        {renderNavigation()}
        <div className="container mx-auto px-4 py-8">
          <LoadingSpinner message="Loading blockchain networks..." />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50">
        {renderNavigation()}
        <div className="container mx-auto px-4 py-8">
          <ErrorMessage message={error} onRetry={loadChains} />
        </div>
      </div>
    );
  }

  if (selectedChain) {
    return (
      <div className="min-h-screen bg-gray-50">
        {renderNavigation()}
        <div className="container mx-auto px-4 py-8">
          <ChainDetail 
            chain={selectedChain} 
            onBack={() => setSelectedChain(null)} 
          />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {renderNavigation()}

      {/* Header */}
      {currentView === 'chains' && (
        <header className="gradient-bg text-white">
          <div className="container mx-auto px-4 py-12">
            <div className="text-center">
              <div className="flex items-center justify-center space-x-3 mb-4">
                <div className="w-12 h-12 bg-white bg-opacity-20 rounded-full flex items-center justify-center">
                  <Globe className="w-6 h-6" />
                </div>
                <h1 className="text-4xl font-bold">Multi-Chain RPC Interface</h1>
              </div>
              <p className="text-xl text-white text-opacity-90 mb-8">
                Comprehensive blockchain network explorer with real-time RPC endpoint testing, smart contract deployment, and Ethereum earning system
              </p>
              
              {/* Stats */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-6 max-w-4xl mx-auto">
                <div className="bg-white bg-opacity-10 rounded-lg p-4">
                  <div className="flex items-center justify-center space-x-2 mb-2">
                    <Globe className="w-5 h-5" />
                    <span className="text-2xl font-bold">{stats.totalChains}</span>
                  </div>
                  <p className="text-sm text-white text-opacity-80">Total Chains</p>
                </div>
                
                <div className="bg-white bg-opacity-10 rounded-lg p-4">
                  <div className="flex items-center justify-center space-x-2 mb-2">
                    <Zap className="w-5 h-5" />
                    <span className="text-2xl font-bold">{stats.totalRpcs}</span>
                  </div>
                  <p className="text-sm text-white text-opacity-80">RPC Endpoints</p>
                </div>
                
                <div className="bg-white bg-opacity-10 rounded-lg p-4">
                  <div className="flex items-center justify-center space-x-2 mb-2">
                    <span className="text-2xl font-bold">{stats.mainnets}</span>
                  </div>
                  <p className="text-sm text-white text-opacity-80">Mainnets</p>
                </div>
                
                <div className="bg-white bg-opacity-10 rounded-lg p-4">
                  <div className="flex items-center justify-center space-x-2 mb-2">
                    <span className="text-2xl font-bold">{stats.testnets}</span>
                  </div>
                  <p className="text-sm text-white text-opacity-80">Testnets</p>
                </div>
              </div>
            </div>
          </div>
        </header>
      )}

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        {currentView === 'earnings' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            <div className="lg:col-span-1">
              <WalletConnection onConnectionChange={setIsWalletConnected} />
            </div>
            <div className="lg:col-span-2">
              <RealTimeEarningsPanel />
            </div>
          </div>
        )}

        {currentView === 'chains' && (
          <>
            <SearchAndFilter
              searchQuery={searchQuery}
              onSearchChange={setSearchQuery}
              sortBy={sortBy}
              onSortChange={setSortBy}
              filters={filters}
              onFilterChange={setFilters}
              totalChains={chains.length}
              filteredChains={filteredAndSortedChains.length}
            />

            {filteredAndSortedChains.length === 0 ? (
              <div className="text-center py-12">
                <Globe className="w-16 h-16 text-gray-300 mx-auto mb-4" />
                <h3 className="text-lg font-semibold text-gray-900 mb-2">No chains found</h3>
                <p className="text-gray-600">Try adjusting your search or filter criteria</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {filteredAndSortedChains.map((chain) => (
                  <ChainCard
                    key={`${chain.chainId}-${chain.name}`}
                    chain={chain}
                    onClick={() => setSelectedChain(chain)}
                  />
                ))}
              </div>
            )}
          </>
        )}

        {currentView === 'deployer' && (
          <SmartContractDeployer chains={chains} />
        )}

        {currentView === 'history' && (
          <DeploymentHistory />
        )}
      </main>

      {/* Footer */}
      <footer className="bg-gray-900 text-white py-8 mt-16">
        <div className="container mx-auto px-4 text-center">
          <p className="text-gray-400">
            Real-time blockchain earnings system with{' '}
            <a 
              href="https://chainlist.org" 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-primary-400 hover:text-primary-300 transition-colors"
            >
              chainlist.org
            </a>
            {' '}integration
          </p>
          <p className="text-sm text-gray-500 mt-2">
            Connect your wallet to start earning ETH from multi-chain operations
          </p>
        </div>
      </footer>
    </div>
  );
}

export default App;