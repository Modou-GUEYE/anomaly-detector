import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TransactionService, Transaction } from '../../services/transaction.service';
import { getCountryNameFromCode, getAllCountriesFR } from '../../utils/country';

// Charger la langue française
// (Import fr retiré: registration gérée dans utilitaire pays)

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
  loading = true;
  error = '';
  expandedRowId: string | null = null;
  // Propriétés inutilisées (selectedTransaction, Math) retirées

  // Filtres
  searchTerm = '';
  filterPays = ''; // Par pays
  filterPeriodStart = ''; // Début période
  filterPeriodEnd = ''; // Fin période
  filterTypeClient = ''; // Type client (Personne/Entreprise)
  filterPortee = ''; // Portée (Nationale/Internationale)
  sortBy: 'date' | 'amount' | 'pays' = 'date';
  // Pagination
  currentPage = 1;
  itemsPerPage = 5;  // 5 transactions par page

  constructor(private transactionService: TransactionService) {}

  ngOnInit() {
    if (typeof window === 'undefined') {
      return;
    }
    this.loadTransactions();
  }

  loadTransactions() {
    this.loading = true;
    this.error = '';
    this.transactionService.getRecentTransactions().subscribe({
      next: (data) => {
        this.transactions = data || [];
        this.applyFilters();
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Erreur lors du chargement des transactions';
        this.loading = false;
        // Log retiré pour éviter bruit en prod; l'erreur est reflétée dans l'UI
      }
    });
  }

  applyFilters() {
    let filtered = [...this.transactions];

    const term = this.searchTerm.trim().toLowerCase();
    const hasSearch = term.length > 0;
    const hasPays = !!this.filterPays;
    const hasPeriodStart = !!this.filterPeriodStart;
    const hasPeriodEnd = !!this.filterPeriodEnd;
    const hasTypeClient = !!this.filterTypeClient;
    const hasPortee = !!this.filterPortee;

    // Pré-calcul dates pour éviter conversions répétées
    let startDate: Date | null = null;
    let endDate: Date | null = null;
    if (hasPeriodStart) startDate = new Date(this.filterPeriodStart);
    if (hasPeriodEnd) {
      endDate = new Date(this.filterPeriodEnd);
      endDate.setHours(23, 59, 59, 999);
    }

    const normalizedPays = hasPays ? normalizeStr(this.filterPays) : '';

    filtered = filtered.filter(t => {
      // Recherche
      if (hasSearch) {
        const matchSearch = (
          t.numero_compte_payeur?.toLowerCase().includes(term) ||
          t.numero_compte_paye?.toLowerCase().includes(term) ||
          t.nom_payeur?.toLowerCase().includes(term) ||
          t.nom_paye?.toLowerCase().includes(term)
        );
        if (!matchSearch) return false;
      }
      // Pays
      if (hasPays) {
        const nomPaysPayeur = this.getCountryNameFromCode(t.pays_payeur);
        const nomPaysPaye = this.getCountryNameFromCode(t.pays_paye);
        const matchPays = normalizeStr(nomPaysPayeur) === normalizedPays || normalizeStr(nomPaysPaye) === normalizedPays;
        if (!matchPays) return false;
      }
      // Période
      if (startDate || endDate) {
        const opDate = new Date(t.date_operation);
        if (startDate && opDate < startDate) return false;
        if (endDate && opDate > endDate) return false;
      }
      // Type client
      if (hasTypeClient) {
        const typeMatch = (
          t.type_client_payeur?.toLowerCase() === this.filterTypeClient.toLowerCase() ||
          t.type_client_paye?.toLowerCase() === this.filterTypeClient.toLowerCase()
        );
        if (!typeMatch) return false;
      }
      // Portée
      if (hasPortee && t.portee !== this.filterPortee) return false;
      return true;
    });

    // Tri factorisé
    const sortStrategies: Record<string, (a: Transaction, b: Transaction) => number> = {
      date: (a, b) => new Date(b.date_operation).getTime() - new Date(a.date_operation).getTime(),
      amount: (a, b) => b.montant - a.montant,
      pays: (a, b) => (a.pays_payeur || '').localeCompare(b.pays_payeur || '')
    };
    const sorter = sortStrategies[this.sortBy] || sortStrategies['date'];
    filtered.sort(sorter);

    this.filteredTransactions = filtered;
    this.currentPage = 1;
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

  get paginatedTransactions(): Transaction[] {
    const start = (this.currentPage - 1) * this.itemsPerPage;
    const end = start + this.itemsPerPage;
    return this.filteredTransactions.slice(start, end);
  }

  get totalPages(): number {
    return Math.ceil(this.filteredTransactions.length / this.itemsPerPage);
  }

  previousPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
    }
  }

  nextPage() {
    if (this.currentPage < this.totalPages) {
      this.currentPage++;
    }
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
    // Restreindre aux pays présents dans les données (optimisation UX)
    const set = new Set<string>();
    this.transactions.forEach(t => {
      if (t.pays_payeur) set.add(this.getCountryNameFromCode(t.pays_payeur));
      if (t.pays_paye) set.add(this.getCountryNameFromCode(t.pays_paye));
    });
    return Array.from(set).sort();
  }

  getUniqueTypeClients(): string[] {
    const types = new Set<string>();
    this.transactions.forEach(t => {
      if (t.type_client_payeur) types.add(t.type_client_payeur);
      if (t.type_client_paye) types.add(t.type_client_paye);
    });
    return Array.from(types).sort();
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
}
