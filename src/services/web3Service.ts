import { ethers } from 'ethers';

export interface WalletConnection {
  address: string;
  provider: ethers.providers.Web3Provider;
  signer: ethers.Signer;
  chainId: number;
  balance: string;
}

export interface RealTimePrice {
  ethereum: {
    usd: number;
    usd_24h_change: number;
  };
}

export interface GasPrice {
  standard: string;
  fast: string;
  instant: string;
}

class Web3Service {
  private connection: WalletConnection | null = null;
  private ethPrice: number = 0;
  private gasPrices: Map<number, GasPrice> = new Map();

  // Cüzdan bağlantısı
  async connectWallet(): Promise<WalletConnection> {
    if (!window.ethereum) {
      throw new Error('MetaMask not installed. Please install MetaMask to continue.');
    }

    try {
      // MetaMask'tan hesap izni iste
      const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
      
      if (!accounts || accounts.length === 0) {
        throw new Error('No accounts found. Please unlock MetaMask.');
      }

      const provider = new ethers.providers.Web3Provider(window.ethereum);
      const signer = provider.getSigner();
      
      // Address'i doğrudan accounts'tan al
      const address = accounts[0];
      
      const network = await provider.getNetwork();
      const balance = await provider.getBalance(address);

      this.connection = {
        address,
        provider,
        signer,
        chainId: network.chainId,
        balance: ethers.utils.formatEther(balance)
      };

      // Hesap değişikliklerini dinle
      window.ethereum.on('accountsChanged', this.handleAccountsChanged.bind(this));
      window.ethereum.on('chainChanged', this.handleChainChanged.bind(this));

      return this.connection;
    } catch (error) {
      console.error('Wallet connection error:', error);
      throw new Error('Failed to connect wallet: ' + (error instanceof Error ? error.message : 'Unknown error'));
    }
  }

  // Cüzdan bağlantısını kes
  async disconnectWallet(): Promise<void> {
    if (window.ethereum) {
      window.ethereum.removeAllListeners();
    }
    this.connection = null;
  }

  // Mevcut bağlantıyı al
  getConnection(): WalletConnection | null {
    return this.connection;
  }

  // Hesap değişikliği
  private async handleAccountsChanged(accounts: string[]): Promise<void> {
    if (accounts.length === 0) {
      this.disconnectWallet();
    } else if (this.connection) {
      this.connection.address = accounts[0];
      // Bakiyeyi güncelle
      await this.updateBalance();
    }
  }

  // Chain değişikliği
  private async handleChainChanged(chainId: string): Promise<void> {
    if (this.connection) {
      this.connection.chainId = parseInt(chainId, 16);
      await this.updateBalance();
    }
  }

  // Bakiyeyi güncelle
  private async updateBalance(): Promise<void> {
    if (!this.connection) return;

    try {
      const balance = await this.connection.provider.getBalance(this.connection.address);
      this.connection.balance = ethers.utils.formatEther(balance);
    } catch (error) {
      console.error('Failed to update balance:', error);
    }
  }

  // Gerçek zamanlı ETH fiyatı
  async fetchRealTimeETHPrice(): Promise<number> {
    try {
      const response = await fetch(
        'https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd&include_24hr_change=true'
      );
      const data: RealTimePrice = await response.json();
      this.ethPrice = data.ethereum.usd;
      return this.ethPrice;
    } catch (error) {
      console.error('Failed to fetch ETH price:', error);
      return this.ethPrice || 3500; // Fallback price
    }
  }

  // Gerçek zamanlı gas fiyatları
  async fetchRealTimeGasPrice(chainId: number): Promise<GasPrice> {
    try {
      let gasPrice: GasPrice;

      if (chainId === 1) { // Ethereum Mainnet
        // Etherscan API kullanmak yerine provider'dan al
        const provider = this.getProviderForChain(chainId);
        const gasPrice_wei = await provider.getGasPrice();
        const gasPrice_gwei = ethers.utils.formatUnits(gasPrice_wei, 'gwei');
        
        gasPrice = {
          standard: gasPrice_gwei,
          fast: (parseFloat(gasPrice_gwei) * 1.2).toString(),
          instant: (parseFloat(gasPrice_gwei) * 1.5).toString()
        };
      } else {
        // Diğer chainler için provider'dan al
        const provider = this.getProviderForChain(chainId);
        const gasPrice_wei = await provider.getGasPrice();
        const gasPrice_gwei = ethers.utils.formatUnits(gasPrice_wei, 'gwei');
        
        gasPrice = {
          standard: gasPrice_gwei,
          fast: (parseFloat(gasPrice_gwei) * 1.2).toString(),
          instant: (parseFloat(gasPrice_gwei) * 1.5).toString()
        };
      }

      this.gasPrices.set(chainId, gasPrice);
      return gasPrice;
    } catch (error) {
      console.error(`Failed to fetch gas price for chain ${chainId}:`, error);
      return { standard: '20', fast: '25', instant: '30' };
    }
  }

  // Chain için provider al
  private getProviderForChain(chainId: number): ethers.providers.JsonRpcProvider {
    const rpcUrls: { [key: number]: string } = {
      1: 'https://eth-mainnet.g.alchemy.com/v2/demo', // Demo key
      137: 'https://polygon-rpc.com',
      56: 'https://bsc-dataseed.binance.org',
      43114: 'https://api.avax.network/ext/bc/C/rpc',
      250: 'https://rpc.ftm.tools',
      42161: 'https://arb1.arbitrum.io/rpc',
      10: 'https://mainnet.optimism.io'
    };

    const rpcUrl = rpcUrls[chainId] || rpcUrls[1];
    return new ethers.providers.JsonRpcProvider(rpcUrl);
  }

  // Transaction gönder
  async sendTransaction(to: string, value: string, data?: string): Promise<string> {
    if (!this.connection) {
      throw new Error('Wallet not connected');
    }

    try {
      const tx = await this.connection.signer.sendTransaction({
        to,
        value: ethers.utils.parseEther(value),
        data: data || '0x'
      });

      return tx.hash;
    } catch (error) {
      throw new Error('Transaction failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
    }
  }

  // Contract deploy
  async deployContract(bytecode: string, abi: any[], constructorArgs: any[] = []): Promise<{
    address: string;
    txHash: string;
    gasUsed: string;
  }> {
    if (!this.connection) {
      throw new Error('Wallet not connected');
    }

    try {
      const factory = new ethers.ContractFactory(abi, bytecode, this.connection.signer);
      const contract = await factory.deploy(...constructorArgs);
      const receipt = await contract.deployTransaction.wait();

      return {
        address: contract.address,
        txHash: contract.deployTransaction.hash,
        gasUsed: receipt.gasUsed.toString()
      };
    } catch (error) {
      throw new Error('Contract deployment failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
    }
  }

  // Transaction receipt al
  async getTransactionReceipt(txHash: string): Promise<ethers.providers.TransactionReceipt | null> {
    if (!this.connection) return null;

    try {
      return await this.connection.provider.getTransactionReceipt(txHash);
    } catch (error) {
      console.error('Failed to get transaction receipt:', error);
      return null;
    }
  }

  // Block numarası al
  async getBlockNumber(): Promise<number> {
    if (!this.connection) return 0;

    try {
      return await this.connection.provider.getBlockNumber();
    } catch (error) {
      console.error('Failed to get block number:', error);
      return 0;
    }
  }

  // Network değiştir
  async switchNetwork(chainId: number): Promise<void> {
    if (!window.ethereum) {
      throw new Error('MetaMask not available');
    }

    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: `0x${chainId.toString(16)}` }],
      });
    } catch (error: any) {
      // Network eklenmemiş ise ekle
      if (error.code === 4902) {
        await this.addNetwork(chainId);
      } else {
        throw error;
      }
    }
  }

  // Network ekle
  private async addNetwork(chainId: number): Promise<void> {
    const networks: { [key: number]: any } = {
      137: {
        chainId: '0x89',
        chainName: 'Polygon Mainnet',
        nativeCurrency: { name: 'MATIC', symbol: 'MATIC', decimals: 18 },
        rpcUrls: ['https://polygon-rpc.com'],
        blockExplorerUrls: ['https://polygonscan.com']
      },
      56: {
        chainId: '0x38',
        chainName: 'BNB Smart Chain',
        nativeCurrency: { name: 'BNB', symbol: 'BNB', decimals: 18 },
        rpcUrls: ['https://bsc-dataseed.binance.org'],
        blockExplorerUrls: ['https://bscscan.com']
      }
    };

    const network = networks[chainId];
    if (!network) {
      throw new Error('Network not supported');
    }

    await window.ethereum.request({
      method: 'wallet_addEthereumChain',
      params: [network],
    });
  }

  // Gerçek zamanlı bakiye takibi
  async startBalanceTracking(callback: (balance: string) => void): Promise<void> {
    if (!this.connection) return;

    const updateBalance = async () => {
      await this.updateBalance();
      if (this.connection) {
        callback(this.connection.balance);
      }
    };

    // İlk güncelleme
    await updateBalance();

    // Her 10 saniyede bir güncelle
    setInterval(updateBalance, 10000);
  }

  // Cüzdan bağlı mı kontrol et
  async isWalletConnected(): Promise<boolean> {
    if (!window.ethereum) return false;

    try {
      const accounts = await window.ethereum.request({ method: 'eth_accounts' });
      return accounts && accounts.length > 0;
    } catch (error) {
      return false;
    }
  }

  // Event listener'ları temizle
  cleanup(): void {
    if (window.ethereum) {
      window.ethereum.removeAllListeners();
    }
  }
}

// Global window interface
declare global {
  interface Window {
    ethereum?: any;
  }
}

export const web3Service = new Web3Service();