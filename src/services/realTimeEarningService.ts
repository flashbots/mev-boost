import { web3Service, WalletConnection } from './web3Service';
import { ethers } from 'ethers';

export interface RealTimeEarning {
  id: string;
  timestamp: Date;
  chainId: number;
  chainName: string;
  operation: 'deployment' | 'transaction' | 'verification';
  txHash: string;
  gasUsed: string;
  gasPrice: string;
  gasCost: string;
  commission: string;
  commissionETH: string;
  userAddress: string;
  blockNumber: number;
  confirmed: boolean;
}

export interface LiveStats {
  totalEarningsETH: string;
  totalEarningsUSD: string;
  todayEarningsETH: string;
  weeklyEarningsETH: string;
  monthlyEarningsETH: string;
  pendingEarningsETH: string;
  totalTransactions: number;
  pendingTransactions: number;
  averageGasPrice: string;
  currentETHPrice: number;
  lastUpdated: Date;
}

class RealTimeEarningService {
  private earnings: RealTimeEarning[] = [];
  private pendingTransactions: Map<string, RealTimeEarning> = new Map();
  private isTracking: boolean = false;
  private updateCallbacks: ((stats: LiveStats) => void)[] = [];
  private commissionRate: number = 2.5; // %2.5

  // Gerçek zamanlı takibi başlat
  async startRealTimeTracking(): Promise<void> {
    if (this.isTracking) return;

    const connection = web3Service.getConnection();
    if (!connection) {
      throw new Error('Wallet not connected');
    }

    this.isTracking = true;

    // Transaction event'lerini dinle
    this.listenToTransactions(connection);

    // Pending transaction'ları kontrol et
    this.startPendingTransactionChecker();

    // ETH fiyatını güncelle
    this.startPriceUpdater();

    console.log('Real-time earning tracking started');
  }

  // Takibi durdur
  stopRealTimeTracking(): void {
    this.isTracking = false;
    this.pendingTransactions.clear();
    console.log('Real-time earning tracking stopped');
  }

  // Transaction event'lerini dinle
  private listenToTransactions(connection: WalletConnection): void {
    // Yeni block'ları dinle
    connection.provider.on('block', async (blockNumber: number) => {
      if (!this.isTracking) return;

      try {
        const block = await connection.provider.getBlockWithTransactions(blockNumber);
        
        // User'ın transaction'larını kontrol et
        for (const tx of block.transactions) {
          if (tx.from?.toLowerCase() === connection.address.toLowerCase()) {
            await this.processTransaction(tx, blockNumber);
          }
        }
      } catch (error) {
        console.error('Error processing block:', error);
      }
    });
  }

  // Transaction'ı işle
  private async processTransaction(tx: any, blockNumber: number): Promise<void> {
    try {
      const receipt = await web3Service.getConnection()?.provider.getTransactionReceipt(tx.hash);
      if (!receipt) return;

      // Gas maliyetini hesapla
      const gasUsed = receipt.gasUsed.toString();
      const gasPrice = tx.gasPrice ? ethers.utils.formatUnits(tx.gasPrice, 'gwei') : '0';
      const gasCost = ethers.utils.formatEther(receipt.gasUsed.mul(tx.gasPrice || 0));

      // Komisyon hesapla
      const commission = parseFloat(gasCost) * (this.commissionRate / 100);
      const commissionETH = commission.toString();

      // Operation type'ını belirle
      let operation: 'deployment' | 'transaction' | 'verification' = 'transaction';
      if (tx.to === null) {
        operation = 'deployment'; // Contract deployment
      } else if (tx.data && tx.data !== '0x') {
        operation = 'verification'; // Contract interaction
      }

      const earning: RealTimeEarning = {
        id: this.generateId(),
        timestamp: new Date(),
        chainId: (await web3Service.getConnection()?.provider.getNetwork())?.chainId || 1,
        chainName: this.getChainName((await web3Service.getConnection()?.provider.getNetwork())?.chainId || 1),
        operation,
        txHash: tx.hash,
        gasUsed,
        gasPrice,
        gasCost,
        commission: commission.toString(),
        commissionETH,
        userAddress: tx.from,
        blockNumber,
        confirmed: true
      };

      // Earning'i kaydet
      this.earnings.push(earning);

      // Pending'den kaldır
      this.pendingTransactions.delete(tx.hash);

      // Callback'leri çağır
      this.notifyUpdates();

      console.log('New earning recorded:', earning);
    } catch (error) {
      console.error('Error processing transaction:', error);
    }
  }

  // Pending transaction ekle
  addPendingTransaction(txHash: string, operation: 'deployment' | 'transaction' | 'verification'): void {
    const connection = web3Service.getConnection();
    if (!connection) return;

    const pendingEarning: RealTimeEarning = {
      id: this.generateId(),
      timestamp: new Date(),
      chainId: connection.chainId,
      chainName: this.getChainName(connection.chainId),
      operation,
      txHash,
      gasUsed: '0',
      gasPrice: '0',
      gasCost: '0',
      commission: '0',
      commissionETH: '0',
      userAddress: connection.address,
      blockNumber: 0,
      confirmed: false
    };

    this.pendingTransactions.set(txHash, pendingEarning);
    this.notifyUpdates();
  }

  // Pending transaction'ları kontrol et
  private startPendingTransactionChecker(): void {
    const checkPending = async () => {
      if (!this.isTracking) return;

      for (const [txHash, earning] of this.pendingTransactions) {
        try {
          const receipt = await web3Service.getTransactionReceipt(txHash);
          if (receipt && receipt.status === 1) {
            // Transaction confirmed
            await this.processTransaction({ hash: txHash, ...receipt }, receipt.blockNumber);
          }
        } catch (error) {
          console.error('Error checking pending transaction:', error);
        }
      }
    };

    // Her 5 saniyede bir kontrol et
    setInterval(checkPending, 5000);
  }

  // ETH fiyatını güncelle
  private startPriceUpdater(): void {
    const updatePrice = async () => {
      if (!this.isTracking) return;

      try {
        await web3Service.fetchRealTimeETHPrice();
        this.notifyUpdates();
      } catch (error) {
        console.error('Error updating ETH price:', error);
      }
    };

    // İlk güncelleme
    updatePrice();

    // Her 30 saniyede bir güncelle
    setInterval(updatePrice, 30000);
  }

  // Canlı istatistikleri al
  getLiveStats(): LiveStats {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
    const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

    // Confirmed earnings
    const confirmedEarnings = this.earnings.filter(e => e.confirmed);
    
    // Total earnings
    const totalEarningsETH = confirmedEarnings
      .reduce((sum, e) => sum + parseFloat(e.commissionETH), 0)
      .toFixed(6);

    // Today earnings
    const todayEarnings = confirmedEarnings
      .filter(e => e.timestamp >= today)
      .reduce((sum, e) => sum + parseFloat(e.commissionETH), 0)
      .toFixed(6);

    // Weekly earnings
    const weeklyEarnings = confirmedEarnings
      .filter(e => e.timestamp >= weekAgo)
      .reduce((sum, e) => sum + parseFloat(e.commissionETH), 0)
      .toFixed(6);

    // Monthly earnings
    const monthlyEarnings = confirmedEarnings
      .filter(e => e.timestamp >= monthAgo)
      .reduce((sum, e) => sum + parseFloat(e.commissionETH), 0)
      .toFixed(6);

    // Pending earnings
    const pendingEarningsETH = Array.from(this.pendingTransactions.values())
      .reduce((sum, e) => sum + parseFloat(e.commissionETH), 0)
      .toFixed(6);

    // Average gas price
    const avgGasPrice = confirmedEarnings.length > 0
      ? (confirmedEarnings.reduce((sum, e) => sum + parseFloat(e.gasPrice), 0) / confirmedEarnings.length).toFixed(2)
      : '0';

    return {
      totalEarningsETH,
      totalEarningsUSD: (parseFloat(totalEarningsETH) * 3500).toFixed(2), // Will be updated with real price
      todayEarningsETH: todayEarnings,
      weeklyEarningsETH: weeklyEarnings,
      monthlyEarningsETH: monthlyEarnings,
      pendingEarningsETH,
      totalTransactions: confirmedEarnings.length,
      pendingTransactions: this.pendingTransactions.size,
      averageGasPrice: avgGasPrice,
      currentETHPrice: 3500, // Will be updated with real price
      lastUpdated: new Date()
    };
  }

  // Update callback ekle
  onStatsUpdate(callback: (stats: LiveStats) => void): void {
    this.updateCallbacks.push(callback);
  }

  // Update callback'lerini çağır
  private notifyUpdates(): void {
    const stats = this.getLiveStats();
    this.updateCallbacks.forEach(callback => callback(stats));
  }

  // Earnings geçmişi
  getEarningsHistory(limit: number = 50): RealTimeEarning[] {
    return [...this.earnings]
      .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
      .slice(0, limit);
  }

  // Pending transactions
  getPendingTransactions(): RealTimeEarning[] {
    return Array.from(this.pendingTransactions.values());
  }

  // Chain name helper
  private getChainName(chainId: number): string {
    const names: { [key: number]: string } = {
      1: 'Ethereum Mainnet',
      137: 'Polygon',
      56: 'BSC',
      43114: 'Avalanche',
      250: 'Fantom',
      42161: 'Arbitrum',
      10: 'Optimism'
    };
    return names[chainId] || `Chain ${chainId}`;
  }

  // ID generator
  private generateId(): string {
    return Date.now().toString(36) + Math.random().toString(36).substr(2);
  }

  // Manual earning ekleme (test için)
  addManualEarning(earning: Omit<RealTimeEarning, 'id' | 'timestamp'>): void {
    this.earnings.push({
      ...earning,
      id: this.generateId(),
      timestamp: new Date()
    });
    this.notifyUpdates();
  }

  // Earnings'leri temizle
  clearEarnings(): void {
    this.earnings = [];
    this.pendingTransactions.clear();
    this.notifyUpdates();
  }
}

export const realTimeEarningService = new RealTimeEarningService();