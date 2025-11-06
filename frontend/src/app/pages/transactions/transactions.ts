import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TransactionService, Transaction } from '../../services/transaction.service';
import * as countries from 'i18n-iso-countries';

// Charger la langue française
import fr from 'i18n-iso-countries/langs/fr.json';

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
  selectedTransaction: Transaction | null = null;
  
  // Accès à Math pour les templates
  Math = Math;

  // Filtres
  searchTerm = '';
  filterPays = ''; // Par pays
  filterPeriodStart = ''; // Début période
  filterPeriodEnd = ''; // Fin période
  filterTypeClient = ''; // Type client (Personne/Entreprise)
  filterPortee = ''; // Portée (Nationale/Internationale)
  sortBy = 'date'; // 'date', 'amount', 'pays'

  // Pagination
  currentPage = 1;
  itemsPerPage = 5;  // 5 transactions par page

  constructor(private transactionService: TransactionService) {}

  ngOnInit() {
    if (typeof window === 'undefined') {
      return;
    }
    // Enregistrer la locale française
    countries.registerLocale(fr);
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
        console.error('Erreur:', err);
      }
    });
  }

  applyFilters() {
    let filtered = this.transactions;

    // Filtre de recherche (numéro compte, nom, etc.)
    if (this.searchTerm) {
      const term = this.searchTerm.toLowerCase();
      filtered = filtered.filter(t =>
        t.numero_compte_payeur?.toLowerCase().includes(term) ||
        t.numero_compte_paye?.toLowerCase().includes(term) ||
        t.nom_payeur?.toLowerCase().includes(term) ||
        t.nom_paye?.toLowerCase().includes(term)
      );
    }

    // Filtre par pays
    if (this.filterPays) {
      const normalizedFilter = normalizeStr(this.filterPays);
      filtered = filtered.filter(t => 
        normalizeStr(t.pays_payeur || '') === normalizedFilter ||
        normalizeStr(t.pays_paye || '') === normalizedFilter
      );
    }

    // Filtre par période
    if (this.filterPeriodStart) {
      const startDate = new Date(this.filterPeriodStart);
      filtered = filtered.filter(t => new Date(t.date_operation) >= startDate);
    }
    if (this.filterPeriodEnd) {
      const endDate = new Date(this.filterPeriodEnd);
      endDate.setHours(23, 59, 59, 999);
      filtered = filtered.filter(t => new Date(t.date_operation) <= endDate);
    }

    // Filtre par type client
    if (this.filterTypeClient) {
      filtered = filtered.filter(t => 
        t.type_client_payeur?.toLowerCase() === this.filterTypeClient.toLowerCase() ||
        t.type_client_paye?.toLowerCase() === this.filterTypeClient.toLowerCase()
      );
    }

    // Filtre par portée
    if (this.filterPortee) {
      filtered = filtered.filter(t => t.portee === this.filterPortee);
    }

    // Tri
    if (this.sortBy === 'date') {
      filtered.sort((a, b) => new Date(b.date_operation).getTime() - new Date(a.date_operation).getTime());
    } else if (this.sortBy === 'amount') {
      filtered.sort((a, b) => b.montant - a.montant);
    } else if (this.sortBy === 'pays') {
      filtered.sort((a, b) => (a.pays_payeur || '').localeCompare(b.pays_payeur || ''));
    }

    this.filteredTransactions = filtered;
    this.currentPage = 1; // Réinitialiser la pagination
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
           transaction.pays_payeur !== transaction.pays_paye ||
           transaction.portee === 'International';
  }

  getRiskLevel(transaction: Transaction): string {
    if (transaction.montant > 1000000) return 'Critique';
    if (transaction.montant > 500000) return 'Élevé';
    if (transaction.pays_payeur !== transaction.pays_paye) return 'Moyen';
    return 'Faible';
  }

  getUniqueCountries(): string[] {
    const countryNames = countries.getNames('fr');
    return Object.values(countryNames).sort();
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
    const total = this.filteredTransactions.length;
    const highRisk = this.filteredTransactions.filter(t => this.isHighRisk(t)).length;
    const international = this.filteredTransactions.filter(t => t.pays_payeur !== t.pays_paye).length;
    const highAmount = this.filteredTransactions.filter(t => t.montant > 500000).length;

    return {
      total,
      highRisk,
      international,
      highAmount,
      totalAmount: this.filteredTransactions.reduce((sum, t) => sum + t.montant, 0),
      averageAmount: total > 0 ? this.filteredTransactions.reduce((sum, t) => sum + t.montant, 0) / total : 0
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
