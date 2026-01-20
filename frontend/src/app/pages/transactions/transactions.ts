import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TransactionService, Transaction } from '../../services/transaction.service';
import { DashboardService } from '../../services/dashboard.service';
import { getCountryNameFromCode, getAllCountriesFR } from '../../utils/country';

// Fonction utilitaire pour normaliser les chaînes (sans accents, minuscule)
function normalizeStr(str: string): string {
  return str.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase();
}

@Component({
  selector: 'app-transactions',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './transactions.html',
  styleUrls: ['./transactions.scss']
})
export class TransactionsComponent implements OnInit {
  transactions: Transaction[] = [];
  filteredTransactions: Transaction[] = [];
  paginatedTransactions: Transaction[] = [];
  loading = true;
  error = '';
  expandedRowId: string | null = null;
  alertes: any[] = [];
  isMobileScreen = false;
  currentPage = 1;
  pageSize = 5;
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

  constructor(private transactionService: TransactionService, private dashboardService: DashboardService) {}

  ngOnInit() {
    if (typeof window === 'undefined') {
      return;
    }
    this.checkScreenSize();
    window.addEventListener('resize', this.checkScreenSize.bind(this));
    this.loadTransactions();
  }

  checkScreenSize() {
    this.isMobileScreen = window.innerWidth <= 768;
  }

  loadTransactions() {
    this.loading = true;
    this.error = '';
    this.dashboardService.getDashboardSummary().subscribe({
      next: (data: any) => {
        this.transactions = data.recent_transactions || [];
        this.alertes = data.recent_anomalies || [];
        this.applyFilters();
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Erreur lors du chargement des transactions';
        this.loading = false;
      }
    });
  }

  applyFilters() {
    const txt = (this.filters.search || '').toLowerCase();
    this.filteredTransactions = this.transactions.filter(t => {
      const okTxt =
        !txt ||
        `${t.nom_payeur} ${t.nom_paye} ${t.numero_compte_payeur} ${t.numero_compte_paye}`
          .toLowerCase()
          .includes(txt);

      let okPays = true;
      if (this.filters.pays) {
        const nomPaysPayeur = this.getCountryNameFromCode(t.pays_payeur);
        const nomPaysPaye = this.getCountryNameFromCode(t.pays_paye);
        const normalizedPaysFilter = normalizeStr(this.filters.pays);
        okPays = 
          normalizeStr(nomPaysPayeur) === normalizedPaysFilter || 
          normalizeStr(nomPaysPaye) === normalizedPaysFilter;
      }

      // Portée
      let okPortee = true;
      if (this.filters.portee) {
        const hasPayeur = !!t.pays_payeur;
        const hasPaye = !!t.pays_paye;
        const isNational = hasPayeur && hasPaye && t.pays_payeur === t.pays_paye;
        const isInternational = hasPayeur && hasPaye && t.pays_payeur !== t.pays_paye;
        const wanted = this.filters.portee.toLowerCase();
        okPortee = (wanted === 'national' && isNational) || (wanted === 'international' && isInternational);
      }

      // Type client (par label lisible)
      let okTypeClient = true;
      if (this.filters.typeClient) {
        const payeurLabel = this.getTypeClientLabel(t.type_client_payeur);
        const payeLabel = this.getTypeClientLabel(t.type_client_paye);
        okTypeClient = payeurLabel === this.filters.typeClient || payeLabel === this.filters.typeClient;
      }

      // Période
      const okPeriode = this.isInSelectedPeriode(t.date_operation);

      return okTxt && okPays && okPortee && okTypeClient && okPeriode;
    });
    this.updatePagination();
  }

  getTypeClientLabel(code: string): string {
    const map: { [key: string]: string } = {
      P: 'Particulier',
      M: 'Entreprise',
      C: 'Commerçant',
      B: 'Business',
    };
    return map[code] || 'Particulier';
  }

  private isInSelectedPeriode(dateOp: string | Date): boolean {
    if (!this.filters.periode) return true;
    const d = new Date(dateOp);
    if (!d) return false;
    const now = new Date();
    const dMidnight = new Date(d.getFullYear(), d.getMonth(), d.getDate());
    const nowMidnight = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const diff = Math.floor((nowMidnight.getTime() - dMidnight.getTime()) / (1000 * 60 * 60 * 24));
    if (this.filters.periode === "Aujourd'hui") return diff === 0;
    if (this.filters.periode === '7 derniers jours') return diff >= 0 && diff <= 7;
    if (this.filters.periode === '30 derniers jours') return diff >= 0 && diff <= 30;
    return true;
  }

  updatePagination() {
    this.totalPages = Math.ceil(this.filteredTransactions.length / this.pageSize);
    if (this.currentPage > this.totalPages && this.totalPages > 0) {
      this.currentPage = this.totalPages;
    }
    const start = (this.currentPage - 1) * this.pageSize;
    const end = start + this.pageSize;
    this.paginatedTransactions = this.filteredTransactions.slice(start, end);
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

  onSearchChange() {
    this.applyFilters();
  }

  onFilterChange() {
    this.applyFilters();
  }

  onSortChange() {
    this.applyFilters();
  }


  toggleRowExpand(transactionId: string) {
    this.expandedRowId = this.expandedRowId === transactionId ? null : transactionId;
  }

  formatMontant(montant: number): string {
    return new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'XOF' }).format(montant);
  }

  formatDate(date: string): string {
    return new Date(date).toLocaleDateString('fr-FR', { 
      day: '2-digit', 
      month: '2-digit', 
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  isHighRisk(transaction: Transaction): boolean {
    return transaction.montant > 500000 || 
           (transaction.pays_payeur && transaction.pays_paye && transaction.pays_payeur !== transaction.pays_paye) ||
           transaction.portee === 'International';
  }

  // Calcule un niveau de risque lisible pour l'UI
  // Retourne l'une de: 'Faible' | 'Moyen' | 'Eleve' (sans accent pour matcher les classes CSS: risk-faible/moyen/eleve)
  getRiskLevel(tx: Transaction): 'Faible' | 'Moyen' | 'Eleve' {
    let score = 0;

    // Montant
    if (tx.montant >= 1_000_000) score += 3;
    else if (tx.montant >= 500_000) score += 2;
    else if (tx.montant >= 200_000) score += 1;

    // Portée / pays
    if (tx.portee === 'International') score += 2;
    if (tx.pays_payeur && tx.pays_paye && tx.pays_payeur !== tx.pays_paye) score += 1;

    // Catégories (si présentes)
    const catPayeur = (tx.categorie_payeur || '').toLowerCase();
    const catPaye = (tx.categorie_paye || '').toLowerCase();
    if (/(pep|interdit|blacklist)/.test(catPayeur) || /(pep|interdit|blacklist)/.test(catPaye)) {
      score += 3;
    } else if (/(surveille|surveillé|surveillee|surveillée)/.test(catPayeur) || /(surveille|surveillé|surveillee|surveillée)/.test(catPaye)) {
      score += 1;
    }

    if (score >= 4) return 'Eleve';
    if (score >= 2) return 'Moyen';
    return 'Faible';
  }

  // Convertir un code pays ISO alpha-2 en nom
  getCountryNameFromCode(code: string): string {
    return getCountryNameFromCode(code);
  }

  getUniqueCountries(): string[] {
    // Retourner tous les pays disponibles (comme dans Comptes)
    return getAllCountriesFR();
  }

  getUniqueTypeClients(): { code: string; label: string }[] {
    return [
      { code: 'P', label: 'Particulier' },
      { code: 'M', label: 'Entreprise' },
      { code: 'C', label: 'Commerçant' },
      { code: 'B', label: 'Business' }
    ];
  }

  getTransactionStats() {
    // Calcul en une seule passe
    let totalAmount = 0;
    let highRisk = 0;
    let international = 0;
    let highAmount = 0;
    for (const t of this.filteredTransactions) {
      totalAmount += t.montant;
      if (this.isHighRisk(t)) highRisk++;
      if (t.pays_payeur !== t.pays_paye) international++;
      if (t.montant > 500000) highAmount++;
    }
    const total = this.filteredTransactions.length;
    return {
      total,
      highRisk,
      international,
      highAmount,
      totalAmount,
      averageAmount: total > 0 ? totalAmount / total : 0
    };
  }

  // Détails de transaction - Expansion/Collapse
  toggleDetails(transaction: Transaction) {
    if (this.expandedRowId === transaction.id) {
      this.expandedRowId = null;
    } else {
      this.expandedRowId = transaction.id;
    }
  }

  isExpanded(transaction: Transaction): boolean {
    return this.expandedRowId === transaction.id;
  }

  // Retourne le statut de la transaction (Succès/Échec) selon la logique dashboard
  getTransactionStatus(tx: Transaction): 'Succès' | 'Échec' {
    if (!tx.id || !this.alertes || this.alertes.length === 0) {
      return 'Succès';
    }
    // Chercher les alertes liées à cette transaction
    const alertesTransaction = this.alertes.filter((alerte: any) => 
      alerte.transactionId === tx.id
    );
    if (alertesTransaction.length === 0) {
      return 'Succès';
    }
    // Vérifier s'il y a des alertes de blocage
    const alertesBlocage = alertesTransaction.filter((alerte: any) => 
      alerte.action && alerte.action.toLowerCase() === 'bloquer'
    );
    if (alertesBlocage.length > 0) {
      return 'Échec';
    }
    // Seul "Interdit" est un échec
    const alertesInterdites = alertesTransaction.filter((alerte: any) => 
      alerte.type === 'Interdit'
    );
    if (alertesInterdites.length > 0) {
      return 'Échec';
    }
    // Blacklist, PEP, Surveillés = transaction validée sous surveillance
    return 'Succès';
  }

  // Retourne la classe CSS du badge statut (comme dashboard)
  getTransactionStatusClass(tx: Transaction): string {
    const status = this.getTransactionStatus(tx);
    return status === 'Succès' ? 'success' : 'failed';
  }
}
