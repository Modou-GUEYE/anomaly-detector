import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

// --- IMPORTS MATERIAL ---
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatButtonModule } from '@angular/material/button';
import { MatTableModule } from '@angular/material/table';

// --- SERVICES ET UTILS ---
import { TransactionService, Transaction } from '../../services/transaction.service';
import { DashboardService } from '../../services/dashboard.service';
import { getCountryNameFromCode, getAllCountriesFR } from '../../utils/country';

// Interface pour les alertes groupées
interface GroupedAlert {
  code: string;
  message: string;
  niveauCriticite: string;
  type: string;
  numeroCompte: string;
  dates: string[];
}

// Fonction utilitaire pour normaliser les chaînes (sans accents, minuscule)
function normalizeStr(str: string): string {
  if (!str) return '';
  return str.normalize('NFD').replaceAll(/\p{Diacritic}/gu, '').toLowerCase();
}

@Component({
  selector: 'app-transactions',
  standalone: true,
  // Ajout des modules Material dans imports pour fixer les erreurs NG8001
  imports: [
    CommonModule, 
    FormsModule, 
    MatIconModule, 
    MatProgressSpinnerModule, 
    MatButtonModule,
    MatTableModule
  ],
  templateUrl: './transactions.html',
  styleUrls: ['./transactions.scss']
})
export class TransactionsComponent implements OnInit, OnDestroy {
  transactions: Transaction[] = [];
  filteredTransactions: Transaction[] = [];
  paginatedTransactions: Transaction[] = [];
  
  loading = true;
  error = '';
  expandedRowId: string | null = null;
  alertes: any[] = [];
  selectedAlertes: any[] = [];
  groupedAlertes: Map<string, GroupedAlert> = new Map(); // Alertes groupées par code
  
  // Modal détails
  showDetailsModal = false;
  detailsTransaction: Transaction | null = null;
  modalAlertes: any[] = [];
  modalGroupedAlertes: Map<string, GroupedAlert> = new Map();
  
  // Pagination
  currentPage = 1;
  pageSize = 10; 
  totalPages = 0;

  filters = {
    search: '',
    pays: '',
    portee: '',
    periode: '',
    typeClient: ''
  };

  typeClientOptions = [
    { code: 'P', label: 'Particulier' },
    { code: 'M', label: 'Entreprise' },
    { code: 'C', label: 'Commerçant' },
    { code: 'B', label: 'Business' },
  ];

  constructor(
    private readonly transactionService: TransactionService,
    private readonly dashboardService: DashboardService
  ) {}

  ngOnInit() {
    if (globalThis.window !== undefined) {
      this.loadTransactions();
    }
  }

  ngOnDestroy() {
    this.expandedRowId = null;
    this.showDetailsModal = false;
    this.detailsTransaction = null;
    this.groupedAlertes.clear();
    this.modalGroupedAlertes.clear();
  }

  loadTransactions() {
    this.loading = true;
    this.error = '';
    
    this.transactionService.getRecentTransactions().subscribe({
      next: (data: Transaction[]) => {
        this.transactions = data || [];
        
        // Charger aussi les alertes du dashboard pour pouvoir les afficher
        this.dashboardService.getDashboardSummary().subscribe({
          next: (dashData: any) => {
            this.alertes = dashData.recent_anomalies || [];
            this.applyFilters();
            this.loading = false;
          },
          error: (err) => {
            console.error('Erreur chargement alertes:', err);
            this.alertes = [];
            this.applyFilters();
            this.loading = false;
          }
        });
      },
      error: (err) => {
        console.error(err);
        this.error = 'Impossible de charger les flux financiers. Veuillez réessayer.';
        this.loading = false;
      }
    });
  }

  applyFilters() {
    const txt = (this.filters.search || '').toLowerCase();
    
    this.filteredTransactions = this.transactions.filter(t => {
      // 1. Recherche textuelle
      const okTxt = !txt || 
        `${t.nom_payeur} ${t.nom_paye} ${t.numero_compte_payeur} ${t.numero_compte_paye}`
          .toLowerCase().includes(txt);

      // 2. Filtre Pays
      let okPays = true;
      if (this.filters.pays) {
        const nomPaysPayeur = this.getCountryNameFromCode(t.pays_payeur);
        const nomPaysPaye = this.getCountryNameFromCode(t.pays_paye);
        const normalizedPaysFilter = normalizeStr(this.filters.pays);
        okPays = normalizeStr(nomPaysPayeur) === normalizedPaysFilter || 
                 normalizeStr(nomPaysPaye) === normalizedPaysFilter;
      }

      // 3. Portée (National vs International)
      let okPortee = true;
      if (this.filters.portee) {
        const isInternational = t.pays_payeur !== t.pays_paye;
        const wanted = this.filters.portee.toLowerCase();
        okPortee = (wanted === 'national' && !isInternational) || 
                   (wanted === 'international' && isInternational);
      }

      // 4. Type Client
      let okTypeClient = true;
      if (this.filters.typeClient) {
        const payeurLabel = this.getTypeClientLabel(t.type_client_payeur);
        const payeLabel = this.getTypeClientLabel(t.type_client_paye);
        okTypeClient = payeurLabel === this.filters.typeClient || 
                       payeLabel === this.filters.typeClient;
      }

      // 5. Période
      const okPeriode = this.isInSelectedPeriode(t.date_operation);

      return okTxt && okPays && okPortee && okTypeClient && okPeriode;
    });

    // Trier du plus récent au plus ancien
    this.filteredTransactions.sort((a, b) => {
      const dateA = new Date(a.date_operation).getTime();
      const dateB = new Date(b.date_operation).getTime();
      return dateB - dateA; // Descendant (plus récent d'abord)
    });

    this.updatePagination();
  }

  // --- LOGIQUE DE PÉRIODE ---
  private isInSelectedPeriode(dateOp: string | Date): boolean {
    if (!this.filters.periode) return true;
    const d = new Date(dateOp);
    const now = new Date();
    const dMidnight = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
    const nowMidnight = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const diffDays = Math.floor((nowMidnight - dMidnight) / (1000 * 60 * 60 * 24));

    switch (this.filters.periode) {
      case "Aujourd'hui": return diffDays === 0;
      case "7 derniers jours": return diffDays >= 0 && diffDays <= 7;
      case "30 derniers jours": return diffDays >= 0 && diffDays <= 30;
      default: return true;
    }
  }

  // --- PAGINATION ---
  updatePagination() {
    this.totalPages = Math.ceil(this.filteredTransactions.length / this.pageSize);
    if (this.currentPage > this.totalPages) this.currentPage = Math.max(1, this.totalPages);
    
    const start = (this.currentPage - 1) * this.pageSize;
    this.paginatedTransactions = this.filteredTransactions.slice(start, start + this.pageSize);
  }

  nextPage() {
    if (this.currentPage < this.totalPages) {
      this.currentPage++;
      this.updatePagination();
    }
  }

  previousPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
      this.updatePagination();
    }
  }

  // --- ACTIONS UI ---
  onSearchChange() { this.currentPage = 1; this.applyFilters(); }
  onFilterChange() { this.currentPage = 1; this.applyFilters(); }

  resetFilters() {
    this.filters = {
      search: '',
      pays: '',
      portee: '',
      periode: '',
      typeClient: ''
    };
    this.currentPage = 1;
    this.applyFilters();
  }

  isFilterActive(): boolean {
    return this.filters.search !== '' || 
           this.filters.pays !== '' || 
           this.filters.portee !== '' || 
           this.filters.periode !== '' || 
           this.filters.typeClient !== '';
  }

  toggleDetails(transaction: Transaction) {
    if (this.expandedRowId === transaction.id) {
      this.expandedRowId = null;
      this.selectedAlertes = [];
      this.groupedAlertes.clear();
    } else {
      this.expandedRowId = transaction.id;
      this.selectedAlertes = this.alertes.filter(a => a.transactionId === transaction.id);
      this.groupAlertes();
    }
  }

  isExpanded(transaction: Transaction): boolean {
    return this.expandedRowId === transaction.id;
  }

  // Grouper les alertes par code avec leurs dates de génération
  groupAlertes() {
    this.groupedAlertes.clear();
    
    for (const alerte of this.selectedAlertes) {
      const code = alerte.code || 'INCONNU';
      const numeroCompte = alerte.numeroCompte || 'N/A';
      const key = `${code}-${numeroCompte}`; // Clé unique: code + compte
      
      if (!this.groupedAlertes.has(key)) {
        this.groupedAlertes.set(key, {
          code: alerte.code,
          message: alerte.message,
          niveauCriticite: alerte.niveauCriticite,
          type: alerte.type,
          numeroCompte: numeroCompte,
          dates: [] as string[]
        });
      }
      
      const grouped = this.groupedAlertes.get(key);
      if (grouped && alerte.dateGeneration) {
        grouped.dates.push(alerte.dateGeneration);
      }
    }
  }

  // Retourner les clés du Map (pour *ngFor)
  getGroupedAlertes(): Array<[string, GroupedAlert]> {
    return Array.from(this.groupedAlertes.entries());
  }

  // Formater une date en français
  formatDateFR(dateStr: string | Date): string {
    if (!dateStr) return '';
    
    const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr;
    if (Number.isNaN(date.getTime())) return '';

    const options: Intl.DateTimeFormatOptions = {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    };

    return new Intl.DateTimeFormat('fr-FR', options).format(date);
  }

  // Formater une date en français sans l'heure (juste jour mois année)
  formatDateFRShort(dateStr: string | Date): string {
    if (!dateStr) return '';
    
    const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr;
    if (Number.isNaN(date.getTime())) return '';

    const options: Intl.DateTimeFormatOptions = {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour12: false
    };

    return new Intl.DateTimeFormat('fr-FR', options).format(date);
  }

  // --- HELPERS DE DONNÉES ---
  getTypeClientLabel(code: string): string {
    const found = this.typeClientOptions.find(opt => opt.code === code);
    return found ? found.label : 'Particulier';
  }

  getCountryNameFromCode(code: string): string {
    return getCountryNameFromCode(code);
  }

  getUniqueCountries(): string[] {
    return getAllCountriesFR();
  }

  getTransactionStatus(tx: Transaction): 'Succès' | 'Échec' {
    if (!this.alertes) return 'Succès';
    const alertesTransaction = this.alertes.filter(a => a.transactionId === tx.id);
    
    const isBlocked = alertesTransaction.some(a => 
      (a.action?.toLowerCase() === 'bloquer') || a.type === 'Interdit'
    );
    
    return isBlocked ? 'Échec' : 'Succès';
  }

  getTransactionStatusClass(tx: Transaction): string {
    return this.getTransactionStatus(tx) === 'Succès' ? 'success' : 'failed';
  }

  getStatusClass(status: string): string {
    switch (status?.toLowerCase()) {
      case 'succès':
        return 'status-succs';
      case 'échec':
        return 'status-chec';
      case 'en attente':
        return 'status-enattente';
      case 'en cours':
        return 'status-encours';
      default:
        return 'status-unknown';
    }
  }

  formatMontant(montant: number): string {
    return new Intl.NumberFormat('fr-FR', { 
      style: 'currency', 
      currency: 'XOF',
      maximumFractionDigits: 0 
    }).format(montant);
  }

  // --- MODAL DÉTAILS ---
  openDetailsModal(transaction: Transaction) {
    this.detailsTransaction = transaction;
    this.modalAlertes = this.alertes.filter(a => a.transactionId === transaction.id);
    this.groupModalAlertes();
    this.showDetailsModal = true;
  }

  closeDetailsModal() {
    this.showDetailsModal = false;
    this.detailsTransaction = null;
    this.modalAlertes = [];
    this.modalGroupedAlertes.clear();
  }

  groupModalAlertes() {
    this.modalGroupedAlertes.clear();
    
    for (const alerte of this.modalAlertes) {
      const code = alerte.code || 'INCONNU';
      const numeroCompte = alerte.numeroCompte || 'N/A';
      const key = `${code}-${numeroCompte}`;
      
      if (!this.modalGroupedAlertes.has(key)) {
        this.modalGroupedAlertes.set(key, {
          code: alerte.code,
          message: alerte.message,
          niveauCriticite: alerte.niveauCriticite,
          type: alerte.type,
          numeroCompte: numeroCompte,
          dates: [] as string[]
        });
      }
      
      const grouped = this.modalGroupedAlertes.get(key);
      if (grouped && alerte.dateGeneration) {
        grouped.dates.push(alerte.dateGeneration);
      }
    }
  }

  getModalGroupedAlertes(): Array<[string, GroupedAlert]> {
    return Array.from(this.modalGroupedAlertes.entries());
  }
}