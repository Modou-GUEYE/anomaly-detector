
import { Component, OnInit } from '@angular/core';
import * as countries from 'i18n-iso-countries';
import fr from 'i18n-iso-countries/langs/fr.json';
// Fonction utilitaire pour normaliser les chaînes (sans accents, minuscule)
function normalizeStr(str: string): string {
  return str.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase();
}
import { DashboardService } from '../../services/dashboard.service';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { HttpClientModule } from '@angular/common/http';
import { FormsModule } from '@angular/forms';

interface Transaction {
  id?: string;
  date_operation: string;
  montant: number;
  portee: string;
  pays_payeur: string;
  ville_payeur: string;
  adresse_payeur: string;
  type_compte_payeur: string;
  type_client_payeur: string;
  categorie_payeur: string;
  numero_compte_payeur: string;
  nom_payeur: string;
  pays_paye: string;
  ville_paye: string;
  adresse_paye: string;
  type_compte_paye: string;
  type_client_paye: string;
  categorie_paye: string;
  numero_compte_paye: string;
  nom_paye: string;
  raw_data?: any;
}

@Component({
  selector: 'app-dashboard',
  standalone: true,
  templateUrl: './dashboard.html',
  styleUrls: ['./dashboard.scss'],
  imports: [CommonModule, FormsModule]
})
export class DashboardComponent implements OnInit {
  error = '';
  loading = true;

  // Données
  transactions: Transaction[] = [];
  filteredTransactions: Transaction[] = [];
  paginatedTransactions: Transaction[] = [];
  alertes: any[] = []; // Alertes pour déterminer le statut des transactions

  // Pagination
  currentPage = 1;
  pageSize = 5;
  totalPages = 0;

  // Recherche globale (header)
  q = '';

  // Filtres locaux
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

  // Mapping code <-> label pour le type de client (doit être dans la classe pour le template)
  TYPE_CLIENT_LABELS: { [key: string]: string } = {
    P: 'Particulier',
    M: 'Entreprise',
    C: 'Commerçant',
    B: 'Business',
  };

  getTypeClientLabel(code: string): string {
    return this.TYPE_CLIENT_LABELS[code] || 'Particulier';
  }

  // Listes pour filtres - Tous les pays du monde en français
  pays: string[] = [];

  // KPI
  kpi = {
    notifications: 3,
    transactions: 0,
    anomaliesDetectees: 0,
    anomaliesTraitees: 0,
    comptesSurveilles: 0,
    montantSuspect: 0,
    frequenceAnormale: 0,
    entitesBlacklistees: 0,
    paysARisque: 0
  };


  constructor(
    private dashboardService: DashboardService,
    private router: Router
  ) {}

  ngOnInit(): void {
    // N'exécuter côté client uniquement
    if (typeof window === 'undefined') {
      return;
    }

    // Enregistrer la locale française pour les pays
    countries.registerLocale(fr);
    
    // Charger la liste complète des pays
    const countryNames = countries.getNames('fr');
    this.pays = Object.values(countryNames).sort();

    // Charger le dashboard avec un seul endpoint optimisé
    this.dashboardService.getDashboardSummary().subscribe({
      next: (data: any) => {
        // KPIs principaux
        this.kpi.transactions = data.transactions_count || 0;
        this.kpi.anomaliesDetectees = data.anomalies_count || 0;
        this.kpi.anomaliesTraitees = data.anomalies_treated || 0;
        this.kpi.comptesSurveilles = data.monitored_accounts || 0;
        
        // Alertes pour déterminer le statut des transactions
        this.alertes = data.recent_anomalies || [];
        
        // Notifications (anomalies non assignées)
        this.kpi.notifications = this.alertes.filter((a: any) => !a.analysteAssigneID).length;

        // Comptages par type d'anomalie
        const byType = data.anomalies_by_type || {};
        this.kpi.montantSuspect = byType['Montant'] || 0;
        this.kpi.frequenceAnormale = byType['frequence'] || 0;
        this.kpi.entitesBlacklistees = byType['Blacklist'] || 0;
        this.kpi.paysARisque = byType['Pays à risque'] || 0;

        // Utiliser les transactions récentes du dashboard summary
        this.transactions = (data.recent_transactions || []).sort((a: any, b: any) => {
          const da = new Date(a.date_operation).getTime();
          const db = new Date(b.date_operation).getTime();
          return db - da;
        });
        
        // Initialiser les filtres et pagination
        this.filters.periode = "Aujourd'hui";
        this.applyFilters();
        this.loading = false;
      },
      error: () => {
        this.error = 'Impossible de charger le résumé du dashboard.';
        this.loading = false;
      }
    });
  }

  // Navigation boutons "Voir détails"
  go(path: string) {
    this.router.navigate(['/', path]);
  }

  // Statuts -> classes CSS
  statusClass(statut?: string) {
    if (!statut || statut.toLowerCase() === 'normal') return 'status-normal';
    if (statut.toLowerCase().includes('avert') || statut.toLowerCase().includes('susp')) return 'status-warning';
    if (statut.toLowerCase().includes('anomal') || statut.toLowerCase().includes('fraude') || statut.toLowerCase().includes('bloqué')) return 'status-alert';
    if (statut.toLowerCase().includes('trait')) return 'status-normal';
    return 'status-normal';
  }

  // Déterminer le statut d'une transaction basé sur les alertes associées
  getTransactionStatus(transaction: Transaction): string {
    if (!transaction.id || !this.alertes || this.alertes.length === 0) {
      return 'Succès';
    }

    // Chercher les alertes liées à cette transaction
    const alertesTransaction = this.alertes.filter(alerte => 
      alerte.transactionId === transaction.id
    );

    if (alertesTransaction.length === 0) {
      return 'Succès';
    }

    // Vérifier s'il y a des alertes de blocage
    const alertesBlocage = alertesTransaction.filter(alerte => 
      alerte.action && alerte.action.toLowerCase() === 'bloquer'
    );

    if (alertesBlocage.length > 0) {
      return 'Échec';
    }

    // Vérifier les alertes critiques (entités interdites)
    const alertesCritiques = alertesTransaction.filter(alerte => 
      alerte.niveauCriticite && alerte.niveauCriticite.toLowerCase() === 'haute' ||
      alerte.type === 'Interdit' || alerte.type === 'Blacklist'
    );

    if (alertesCritiques.length > 0) {
      return 'Échec';
    }

    return 'Succès';
  }

  // Classe CSS pour le statut de transaction
  getTransactionStatusClass(transaction: Transaction): string {
    const status = this.getTransactionStatus(transaction);
    return status === 'Succès' ? 'status-success' : 'status-failure';
  }

  // Filtrage local (appelle ceci sur (ngModelChange) si tu veux filtrage live)
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
        const normalizedPaysPayeur = t.pays_payeur ? normalizeStr(t.pays_payeur) : '';
        const normalizedPaysPaye = t.pays_paye ? normalizeStr(t.pays_paye) : '';
        const normalizedPaysFilter = normalizeStr(this.filters.pays);
        okPays = normalizedPaysPayeur === normalizedPaysFilter || normalizedPaysPaye === normalizedPaysFilter;
      }

      const okPortee = !this.filters.portee ||
        (t.portee && t.portee.toLowerCase() === this.filters.portee.toLowerCase());

      // Filtre type de client (par label lisible)
      let okTypeClient = true;
      if (this.filters.typeClient) {
        // On filtre si le type client payeur OU bénéficiaire correspond au label sélectionné
        const payeurLabel = this.TYPE_CLIENT_LABELS[t.type_client_payeur] || 'Particulier';
        const payeLabel = this.TYPE_CLIENT_LABELS[t.type_client_paye] || 'Particulier';
        okTypeClient = payeurLabel === this.filters.typeClient || payeLabel === this.filters.typeClient;
      }

      // Période simple (exemple)
      const okPeriode = (() => {
        if (!this.filters.periode) return true;
        
        // Parse date_operation properly, handling both ISO strings and other formats
        let d: Date;
        if (typeof t.date_operation === 'string') {
          // If it's an ISO string like "2025-10-22", parse it in local timezone
          if (t.date_operation.includes('T')) {
            d = new Date(t.date_operation); // Already has time component
          } else {
            // Format like "2025-10-22" - add noon to avoid timezone shift
            d = new Date(t.date_operation + 'T12:00:00');
          }
        } else {
          d = new Date(t.date_operation);
        }
        
        const now = new Date();
        
        // Compare dates at midnight for accurate day comparison
        const dMidnight = new Date(d.getFullYear(), d.getMonth(), d.getDate());
        const nowMidnight = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        
        const diff = (nowMidnight.getTime() - dMidnight.getTime()) / (1000 * 60 * 60 * 24);
        
        if (this.filters.periode === "Aujourd'hui") return diff === 0;
        if (this.filters.periode === '7 derniers jours') return diff >= 0 && diff <= 7;
        if (this.filters.periode === '30 derniers jours') return diff >= 0 && diff <= 30;
        return true;
      })();

      return okTxt && okPays && okPortee && okTypeClient && okPeriode;
    });
    this.updatePagination();
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

  prevPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
      this.updatePagination();
    }
  }

  goToPage(page: number) {
    if (page >= 1 && page <= this.totalPages) {
      this.currentPage = page;
      this.updatePagination();
    }
  }
}