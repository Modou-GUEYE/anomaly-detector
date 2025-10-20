

import { Component, OnInit } from '@angular/core';
// Fonction utilitaire pour normaliser les chaînes (sans accents, minuscule)
function normalizeStr(str: string): string {
  return str.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase();
}
import { ProfileService } from '../../services/profile.service';
import { AlertService } from '../../services/alert.service';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

import { AnomalyChartComponent } from '../../components/anomaly-chart/anomaly-chart';
import { NgChartsModule } from 'ng2-charts';
import { TransactionService, Transaction } from '../../services/transaction.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  templateUrl: './dashboard.html',
  styleUrls: ['./dashboard.scss'],
  imports: [CommonModule, FormsModule, AnomalyChartComponent, NgChartsModule]
})
export class DashboardComponent implements OnInit {
  // Renvoie la classe CSS pour le statut (toujours normal ici)
  onLogout(): void {
    if (typeof window !== 'undefined' && window.localStorage) {
      localStorage.removeItem('token');
    }
    this.router.navigate(['/login']);
  }
  profile: any = null;
  error = '';
  loading = true;

  // Données
  transactions: Transaction[] = [];
  filteredTransactions: Transaction[] = [];

  // Recherche globale (header)
  q = '';

  // Filtres locaux
  filters = {
    search: '',
    pays: '',
    portee: '',
    periode: '',
    typeClient: '' // label lisible (ex: 'Particulier')
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

  // Listes pour filtres
  pays: string[] = [
    'Bénin',
    'Burkina Faso',
    'Côte d’Ivoire',
    'Guinée-Bissau',
    'Mali',
    'Niger',
    'Sénégal',
    'Togo'
  ];

  // Onglets anomalies
  activeTab: 'overview' | 'type' | 'criticite' = 'overview';

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
    private profileService: ProfileService,
    private transactionService: TransactionService,
    private alertService: AlertService,
    private router: Router
  ) {}

  ngOnInit(): void {
    // Charger profil
    this.profileService.getProfile().subscribe({
      next: (data) => { this.profile = data; },
      error: () => { this.error = 'Impossible de charger le profil.'; }
    });

    // Charger transactions
    this.transactionService.getRecentTransactions().subscribe({
      next: (data) => {
        // Tri décroissant par date et heure
        this.transactions = (data || []).sort((a, b) => {
          const da = new Date(a.date_operation).getTime();
          const db = new Date(b.date_operation).getTime();
          return db - da;
        });
        this.filteredTransactions = [...this.transactions];

        // KPI de base (exemples) – adapte selon ton backend
        this.kpi.transactions = this.transactions.length;
        this.kpi.comptesSurveilles = new Set(this.transactions.map(t => t.numero_compte_payeur)).size;
      },
      error: () => { this.error = 'Impossible de charger les transactions.'; },
      complete: () => { this.loading = false; }
    });

    // Charger alertes (anomalies)
    this.alertService.getAlertes().subscribe({
      next: (alertes) => {
        this.kpi.anomaliesDetectees = alertes.length;
        this.kpi.anomaliesTraitees = alertes.filter(a => a.statut && a.statut.toLowerCase().includes('trait')).length;

        // Comptage par type d'anomalie
        this.kpi.montantSuspect = alertes.filter(a => a.type && a.type.toLowerCase().includes('montant')).length;
        this.kpi.frequenceAnormale = alertes.filter(a => a.type && a.type.toLowerCase().includes('fréquence')).length;
        this.kpi.entitesBlacklistees = alertes.filter(a => a.type && a.type.toLowerCase().includes('blacklist')).length;
        this.kpi.paysARisque = alertes.filter(a => a.type && a.type.toLowerCase().includes('risque')).length;
      },
      error: () => { this.error = 'Impossible de charger les alertes.'; }
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
        console.log('Pays payeur:', t.pays_payeur, '| Normalized:', normalizedPaysPayeur);
        console.log('Pays bénéficiaire:', t.pays_paye, '| Normalized:', normalizedPaysPaye);
        console.log('Pays filter:', this.filters.pays, '| Normalized:', normalizedPaysFilter);
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
        const d = new Date(t.date_operation);
        const now = new Date();
        const diff = (now.getTime() - d.getTime()) / (1000 * 60 * 60 * 24);
        if (this.filters.periode === 'Aujourd’hui') return d.toDateString() === now.toDateString();
        if (this.filters.periode === '7 derniers jours') return diff <= 7;
        if (this.filters.periode === '30 derniers jours') return diff <= 30;
        return true;
      })();

      return okTxt && okPays && okPortee && okTypeClient && okPeriode;
    });
  }

  // Utilitaire pour afficher le label lisible dans le tableau
}
