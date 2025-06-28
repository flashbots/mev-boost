import { ChainData } from '../types/chain';

export interface EarningConfig {
  commissionRate: number; // Yüzde olarak komisyon oranı (örn: 2.5 = %2.5)
  minimumEarning: string; // Minimum kazanç miktarı (ETH)
  autoWithdraw: boolean; // Otomatik çekim
  withdrawThreshold: string; // Çekim eşiği (ETH)
  treasuryAddress: string; // Ana cüzdan adresi
}

export interface EarningRecord {
  id: string;
  timestamp: Date;
  chainId: number;
  chainName: string;
  operation: 'deployment' | 'transaction' | 'verification' | 'interaction';
  gasUsed: string;
  gasPaid: string;
  commission: string;
  commissionInETH: string;
  txHash: string;
  userAddress: string;
}

export interface EarningStats {
  totalEarningsETH: string;
  totalEarningsUSD: string;
  todayEarningsETH: string;
  weeklyEarningsETH: string;
  monthlyEarningsETH: string;
  totalTransactions: number;
  averageCommission: string;
  topEarningChains: Array<{
    chainId: number;
    chainName: string;
    earnings: string;
    percentage: number;
  }>;
}

class EthereumEarningService {
  private config: EarningConfig = {
    commissionRate: 2.5, // %2.5 komisyon
    minimumEarning: '0.001', // Minimum 0.001 ETH
    autoWithdraw: true,
    withdrawThreshold: '1.0', // 1 ETH'de otomatik çekim
    treasuryAddress: '0x742d35Cc6634C0532925a3b8D4C9db96c4b4d8b6' // Ana cüzdan
  };

  private earnings: EarningRecord[] = [];
  private ethPrice: number = 3500; // USD cinsinden ETH fiyatı

  // Komisyon hesaplama
  calculateCommission(gasUsed: string, gasPrice: string, chainId: number): {
    commission: string;
    commissionInETH: string;
  } {
    const gasCost = parseFloat(gasUsed) * parseFloat(gasPrice);
    const commission = gasCost * (this.config.commissionRate / 100);
    
    // Chain'e göre ETH'e çevirme oranı
    const ethConversionRate = this.getETHConversionRate(chainId);
    const commissionInETH = commission * ethConversionRate;

    return {
      commission: commission.toString(),
      commissionInETH: commissionInETH.toString()
    };
  }

  // Chain'e göre ETH çevirme oranı
  private getETHConversionRate(chainId: number): number {
    const rates: { [key: number]: number } = {
      1: 1.0,      // Ethereum Mainnet
      137: 0.0015, // Polygon (MATIC to ETH)
      56: 0.0065,  // BSC (BNB to ETH)
      43114: 0.045, // Avalanche (AVAX to ETH)
      250: 0.0008, // Fantom (FTM to ETH)
      42161: 1.0,  // Arbitrum (ETH)
      10: 1.0,     // Optimism (ETH)
      100: 0.0035, // Gnosis (xDAI to ETH)
    };
    
    return rates[chainId] || 0.001; // Default düşük oran
  }

  // Deployment'tan kazanç kaydetme
  async recordDeploymentEarning(
    chainId: number,
    chainName: string,
    gasUsed: string,
    gasPrice: string,
    txHash: string,
    userAddress: string
  ): Promise<EarningRecord> {
    const { commission, commissionInETH } = this.calculateCommission(gasUsed, gasPrice, chainId);
    
    const earning: EarningRecord = {
      id: this.generateId(),
      timestamp: new Date(),
      chainId,
      chainName,
      operation: 'deployment',
      gasUsed,
      gasPaid: (parseFloat(gasUsed) * parseFloat(gasPrice)).toString(),
      commission,
      commissionInETH,
      txHash,
      userAddress
    };

    this.earnings.push(earning);
    
    // Otomatik çekim kontrolü
    if (this.config.autoWithdraw) {
      await this.checkAutoWithdraw();
    }

    return earning;
  }

  // Transaction'dan kazanç kaydetme
  async recordTransactionEarning(
    chainId: number,
    chainName: string,
    gasUsed: string,
    gasPrice: string,
    txHash: string,
    userAddress: string,
    operation: 'transaction' | 'verification' | 'interaction' = 'transaction'
  ): Promise<EarningRecord> {
    const { commission, commissionInETH } = this.calculateCommission(gasUsed, gasPrice, chainId);
    
    const earning: EarningRecord = {
      id: this.generateId(),
      timestamp: new Date(),
      chainId,
      chainName,
      operation,
      gasUsed,
      gasPaid: (parseFloat(gasUsed) * parseFloat(gasPrice)).toString(),
      commission,
      commissionInETH,
      txHash,
      userAddress
    };

    this.earnings.push(earning);
    
    if (this.config.autoWithdraw) {
      await this.checkAutoWithdraw();
    }

    return earning;
  }

  // Otomatik çekim kontrolü
  private async checkAutoWithdraw(): Promise<void> {
    const totalEarnings = this.getTotalEarningsETH();
    
    if (parseFloat(totalEarnings) >= parseFloat(this.config.withdrawThreshold)) {
      await this.withdrawEarnings();
    }
  }

  // Kazançları çekme
  async withdrawEarnings(): Promise<{
    success: boolean;
    txHash?: string;
    amount: string;
    error?: string;
  }> {
    try {
      const totalAmount = this.getTotalEarningsETH();
      
      if (parseFloat(totalAmount) < parseFloat(this.config.minimumEarning)) {
        return {
          success: false,
          amount: totalAmount,
          error: 'Amount below minimum withdrawal threshold'
        };
      }

      // Simulated withdrawal - gerçek uygulamada blockchain transaction'ı yapılır
      const txHash = await this.simulateWithdrawal(totalAmount);
      
      // Earnings'leri temizle (çekildi olarak işaretle)
      this.markEarningsAsWithdrawn();

      return {
        success: true,
        txHash,
        amount: totalAmount
      };

    } catch (error) {
      return {
        success: false,
        amount: '0',
        error: error instanceof Error ? error.message : 'Withdrawal failed'
      };
    }
  }

  // Withdrawal simülasyonu
  private async simulateWithdrawal(amount: string): Promise<string> {
    // Gerçek uygulamada Web3/Ethers.js ile transaction yapılır
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    // Mock transaction hash
    return '0x' + Array.from({length: 64}, () => 
      Math.floor(Math.random() * 16).toString(16)
    ).join('');
  }

  // Earnings'leri çekildi olarak işaretle
  private markEarningsAsWithdrawn(): void {
    // Gerçek uygulamada database'de withdrawn flag set edilir
    this.earnings = [];
  }

  // Toplam ETH kazancı
  getTotalEarningsETH(): string {
    const total = this.earnings.reduce((sum, earning) => 
      sum + parseFloat(earning.commissionInETH), 0
    );
    return total.toFixed(6);
  }

  // Toplam USD kazancı
  getTotalEarningsUSD(): string {
    const totalETH = parseFloat(this.getTotalEarningsETH());
    return (totalETH * this.ethPrice).toFixed(2);
  }

  // Günlük kazanç
  getTodayEarningsETH(): string {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    const todayEarnings = this.earnings.filter(earning => 
      earning.timestamp >= today
    );
    
    const total = todayEarnings.reduce((sum, earning) => 
      sum + parseFloat(earning.commissionInETH), 0
    );
    
    return total.toFixed(6);
  }

  // Haftalık kazanç
  getWeeklyEarningsETH(): string {
    const weekAgo = new Date();
    weekAgo.setDate(weekAgo.getDate() - 7);
    
    const weeklyEarnings = this.earnings.filter(earning => 
      earning.timestamp >= weekAgo
    );
    
    const total = weeklyEarnings.reduce((sum, earning) => 
      sum + parseFloat(earning.commissionInETH), 0
    );
    
    return total.toFixed(6);
  }

  // Aylık kazanç
  getMonthlyEarningsETH(): string {
    const monthAgo = new Date();
    monthAgo.setMonth(monthAgo.getMonth() - 1);
    
    const monthlyEarnings = this.earnings.filter(earning => 
      earning.timestamp >= monthAgo
    );
    
    const total = monthlyEarnings.reduce((sum, earning) => 
      sum + parseFloat(earning.commissionInETH), 0
    );
    
    return total.toFixed(6);
  }

  // En çok kazandıran chain'ler
  getTopEarningChains(): Array<{
    chainId: number;
    chainName: string;
    earnings: string;
    percentage: number;
  }> {
    const chainEarnings = new Map<number, { name: string; total: number }>();
    
    this.earnings.forEach(earning => {
      const current = chainEarnings.get(earning.chainId) || { name: earning.chainName, total: 0 };
      current.total += parseFloat(earning.commissionInETH);
      chainEarnings.set(earning.chainId, current);
    });

    const totalEarnings = parseFloat(this.getTotalEarningsETH());
    
    return Array.from(chainEarnings.entries())
      .map(([chainId, data]) => ({
        chainId,
        chainName: data.name,
        earnings: data.total.toFixed(6),
        percentage: totalEarnings > 0 ? (data.total / totalEarnings) * 100 : 0
      }))
      .sort((a, b) => parseFloat(b.earnings) - parseFloat(a.earnings))
      .slice(0, 10);
  }

  // İstatistikler
  getEarningStats(): EarningStats {
    return {
      totalEarningsETH: this.getTotalEarningsETH(),
      totalEarningsUSD: this.getTotalEarningsUSD(),
      todayEarningsETH: this.getTodayEarningsETH(),
      weeklyEarningsETH: this.getWeeklyEarningsETH(),
      monthlyEarningsETH: this.getMonthlyEarningsETH(),
      totalTransactions: this.earnings.length,
      averageCommission: this.getAverageCommission(),
      topEarningChains: this.getTopEarningChains()
    };
  }

  // Ortalama komisyon
  private getAverageCommission(): string {
    if (this.earnings.length === 0) return '0';
    
    const total = this.earnings.reduce((sum, earning) => 
      sum + parseFloat(earning.commissionInETH), 0
    );
    
    return (total / this.earnings.length).toFixed(6);
  }

  // Konfigürasyon güncelleme
  updateConfig(newConfig: Partial<EarningConfig>): void {
    this.config = { ...this.config, ...newConfig };
  }

  // Konfigürasyon alma
  getConfig(): EarningConfig {
    return { ...this.config };
  }

  // ETH fiyatı güncelleme
  async updateETHPrice(): Promise<void> {
    try {
      // Gerçek uygulamada CoinGecko, CoinMarketCap vb. API'lerden fiyat alınır
      const response = await fetch('https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd');
      const data = await response.json();
      this.ethPrice = data.ethereum.usd;
    } catch (error) {
      console.error('Failed to update ETH price:', error);
    }
  }

  // Earnings geçmişi
  getEarningsHistory(limit: number = 100): EarningRecord[] {
    return this.earnings
      .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
      .slice(0, limit);
  }

  // Belirli bir chain'den kazançlar
  getEarningsByChain(chainId: number): EarningRecord[] {
    return this.earnings.filter(earning => earning.chainId === chainId);
  }

  // Belirli bir kullanıcıdan kazançlar
  getEarningsByUser(userAddress: string): EarningRecord[] {
    return this.earnings.filter(earning => 
      earning.userAddress.toLowerCase() === userAddress.toLowerCase()
    );
  }

  // ID generator
  private generateId(): string {
    return Date.now().toString(36) + Math.random().toString(36).substr(2);
  }

  // Mock earnings data for demo
  generateMockEarnings(): void {
    const mockChains = [
      { id: 1, name: 'Ethereum Mainnet' },
      { id: 137, name: 'Polygon' },
      { id: 56, name: 'BSC' },
      { id: 43114, name: 'Avalanche' },
      { id: 250, name: 'Fantom' }
    ];

    // Son 30 gün için mock data
    for (let i = 0; i < 150; i++) {
      const chain = mockChains[Math.floor(Math.random() * mockChains.length)];
      const daysAgo = Math.floor(Math.random() * 30);
      const timestamp = new Date();
      timestamp.setDate(timestamp.getDate() - daysAgo);

      const gasUsed = (Math.random() * 500000 + 100000).toString();
      const gasPrice = (Math.random() * 50 + 10).toString();
      const { commission, commissionInETH } = this.calculateCommission(gasUsed, gasPrice, chain.id);

      this.earnings.push({
        id: this.generateId(),
        timestamp,
        chainId: chain.id,
        chainName: chain.name,
        operation: ['deployment', 'transaction', 'verification'][Math.floor(Math.random() * 3)] as any,
        gasUsed,
        gasPaid: (parseFloat(gasUsed) * parseFloat(gasPrice)).toString(),
        commission,
        commissionInETH,
        txHash: '0x' + Array.from({length: 64}, () => Math.floor(Math.random() * 16).toString(16)).join(''),
        userAddress: '0x' + Array.from({length: 40}, () => Math.floor(Math.random() * 16).toString(16)).join('')
      });
    }
  }
}

export const ethereumEarningService = new EthereumEarningService();