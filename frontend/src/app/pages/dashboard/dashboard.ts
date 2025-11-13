
import { Component, OnInit, HostListener } from '@angular/core';
import { getCountryNameFromCode as countryNameFromCode, getAllCountriesFR } from '../../utils/country';
// Fonction utilitaire pour normaliser les chaînes (sans accents, minuscule)
function normalizeStr(str: string): string {
  return str.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase();
}
import { DashboardService } from '../../services/dashboard.service';
import { TransactionService } from '../../services/transaction.service';
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
  // allow indexed access from templates for legacy keys coming from API
  [key: string]: any;
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
  selectedTransaction: Transaction | null = null;
  selectedAlertes: any[] = [];
  // pour restauration du focus après fermeture du modal
  private _lastFocusedElement: Element | null = null;

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

  // Ouvrir le détail d'une transaction (modal)
  viewTransaction(t: Transaction) {
    this.selectedTransaction = t;
    this.selectedAlertes = this.alertes.filter(a => a.transactionId === t.id);
    if (typeof document !== 'undefined') {
      // mémoriser l'élément focussé avant d'ouvrir la modal
      this._lastFocusedElement = document.activeElement;
      document.body.classList.add('modal-open');
      // donner le focus au premier bouton/action dans la modal pour accessibilité
      setTimeout(() => {
        const el = document.querySelector('.modal-content .card-btn, .modal-content .close-btn, .modal-content button') as HTMLElement | null;
        if (el) { el.focus(); }
      }, 0);
    }
  }

  // Fermer la modal de détail
  closeTransactionDetail() {
    this.selectedTransaction = null;
    this.selectedAlertes = [];
    if (typeof document !== 'undefined') {
      document.body.classList.remove('modal-open');
      // restaurer le focus à l'élément précédent
      try {
        const last = this._lastFocusedElement as HTMLElement | null;
        if (last && typeof last.focus === 'function') {
          last.focus();
        }
      } catch (e) {
        // silent
      }
    }
  }

  // Statut lisible combiné
  getReadableStatus(): string {
    if (!this.selectedTransaction) return '';
    return this.getTransactionStatus(this.selectedTransaction);
  }

  // Fermer avec la touche Échap
  @HostListener('document:keydown.escape', ['$event'])
  onEscKey(_: Event) {
    if (this.selectedTransaction) {
      this.closeTransactionDetail();
    }
  }

  // Portée calculée à partir des pays
  getTransactionScope(t: Transaction): string {
    if (!t.pays_payeur || !t.pays_paye) return '';
    return t.pays_payeur === t.pays_paye ? 'National' : 'International';
  }

  // Listes pour filtres - Tous les pays du monde en français
  pays: string[] = [];

  // KPI
  kpi = {
    notifications: 0,
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
    private transactionService: TransactionService,
    private router: Router
  ) {}

  ngOnInit(): void {
    // N'exécuter côté client uniquement
    if (typeof window === 'undefined') {
      return;
    }

    // Définir le titre de la page pour l'onglet navigateur
    if (typeof document !== 'undefined') {
      document.title = 'Métriques du Tableau de Bord - Anomaly Detector';
    }

  // Charger la liste complète des pays en français via utilitaire partagé
  this.pays = getAllCountriesFR();

    // Charger le dashboard (réutilisable depuis refresh)
    this.loadSummary();
  }

  // Extraire la logique de chargement pour pouvoir rafraîchir depuis l'UI
  loadSummary() {
    this.loading = true;
    this.error = '';
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

        // Utiliser les transactions récentes du dashboard summary si elles existent
        const summaryTx = (data.recent_transactions || []);

        // Si le summary ne fournit pas toutes les transactions (typiquement limité à 10),
        // récupérer la liste complète via le service Transactions pour alimenter le tableau.
        const setTransactions = (txs: any[]) => {
          this.transactions = (txs || []).sort((a: any, b: any) => {
            const da = new Date(a.date_operation).getTime();
            const db = new Date(b.date_operation).getTime();
            return db - da;
          });
          // Initialiser les filtres et pagination
          this.filters.periode = "Aujourd'hui";
          this.applyFilters();
          this.loading = false;
        };

        if (summaryTx.length >= 50) {
          // Si le summary contient déjà suffisamment d'éléments, on l'utilise
          setTransactions(summaryTx);
        } else {
          // Sinon, récupérer la liste complète des transactions disponible via l'API
          this.transactionService.getRecentTransactions().subscribe({
            next: (fullTxs: any[]) => setTransactions(fullTxs.length ? fullTxs : summaryTx),
            error: () => {
              // En cas d'erreur, fallback sur ce que le summary fournit
              setTransactions(summaryTx);
            }
          });
        }
      },
      error: () => {
        this.error = 'Impossible de charger le résumé du dashboard.';
        this.loading = false;
      }
    });
  }

  // Méthode appelée par le bouton Refresh de l'UI
  refreshData() {
    this.loadSummary();
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

  // Convertir un code pays ISO alpha-2 en nom (ex: "SN" → "Sénégal") avec guard ISO (5)
  getCountryNameFromCode(code: string): string {
    return countryNameFromCode(code);
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
        // Convertir les codes pays des transactions en noms et comparer avec le filtre
        const nomPaysPayeur = this.getCountryNameFromCode(t.pays_payeur);
        const nomPaysPaye = this.getCountryNameFromCode(t.pays_paye);
        const normalizedPaysFilter = normalizeStr(this.filters.pays);
        
        okPays = 
          normalizeStr(nomPaysPayeur) === normalizedPaysFilter || 
          normalizeStr(nomPaysPaye) === normalizedPaysFilter;
      }

      // Filtre portée: calculer dynamiquement à partir des pays (les données récentes n'ont pas toujours t.portee)
      let okPortee = true;
      if (this.filters.portee) {
        const hasPayeur = !!t.pays_payeur;
        const hasPaye = !!t.pays_paye;
        const isNational = hasPayeur && hasPaye && t.pays_payeur === t.pays_paye;
        const isInternational = hasPayeur && hasPaye && t.pays_payeur !== t.pays_paye;
        const wanted = this.filters.portee.toLowerCase();
        okPortee = (wanted === 'national' && isNational) || (wanted === 'international' && isInternational);
      }

      // Filtre type de client (par label lisible)
      let okTypeClient = true;
      if (this.filters.typeClient) {
        // On filtre si le type client payeur OU bénéficiaire correspond au label sélectionné
        const payeurLabel = this.TYPE_CLIENT_LABELS[t.type_client_payeur] || 'Particulier';
        const payeLabel = this.TYPE_CLIENT_LABELS[t.type_client_paye] || 'Particulier';
        okTypeClient = payeurLabel === this.filters.typeClient || payeLabel === this.filters.typeClient;
      }

      // Période (4) déléguée à une fonction dédiée
      const okPeriode = this.isInSelectedPeriode(t.date_operation);

      return okTxt && okPays && okPortee && okTypeClient && okPeriode;
    });
    this.updatePagination();
  }

  // (4) Extraction de la logique de période
  private isInSelectedPeriode(dateOp: string | Date): boolean {
    if (!this.filters.periode) return true;
    const d = this.parseDateOp(dateOp);
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

  // Parse API date string into a Date object; return null on failure.
  private parseDateOp(input: string | Date): Date | null {
    if (input instanceof Date) return input;
    const s = (input || '').toString().trim();

    // Try to parse a strict ISO-like timestamp with optional fractional seconds and timezone
    // Examples supported:
    //  - 2025-11-10T14:38:52.329452Z
    //  - 2025-11-04 13:15:19.172354+00
    //  - 2025-11-04T13:15:19+01:00
    const re = /^([0-9]{4})-([0-9]{2})-([0-9]{2})(?:[T\s])([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.(\d+))?(Z|([+-])(\d{2}):?(\d{2}))?$/;
    const m = s.match(re);
    if (!m) return null;

    const year = Number(m[1]);
    const month = Number(m[2]) - 1;
    const day = Number(m[3]);
    const hour = Number(m[4]);
    const minute = Number(m[5]);
    const second = Number(m[6]);
    let fraction = m[7] || '';
    // Normalize fraction to milliseconds
    let ms = 0;
    if (fraction) {
      if (fraction.length > 3) fraction = fraction.slice(0, 3);
      while (fraction.length < 3) fraction += '0';
      ms = Number(fraction);
    }

    // Timezone handling
    let tzOffsetMinutes = 0; // minutes to subtract to get UTC
    if (m[8] === 'Z' || m[8] === undefined) {
      tzOffsetMinutes = 0;
    } else if (m[9] === '+' || m[9] === '-') {
      const sign = m[9] === '+' ? 1 : -1;
      const tzHour = Number(m[10] || '0');
      const tzMin = Number(m[11] || '0');
      tzOffsetMinutes = sign * (tzHour * 60 + tzMin);
      // Note: to get UTC from local, subtract the offset minutes
    }

    // Build UTC timestamp then apply timezone offset
    const utcMillis = Date.UTC(year, month, day, hour, minute, second, ms);
    const final = new Date(utcMillis - tzOffsetMinutes * 60 * 1000);
    if (isNaN(final.getTime())) return null;
    return final;
  }

  // (removed debug helper)

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