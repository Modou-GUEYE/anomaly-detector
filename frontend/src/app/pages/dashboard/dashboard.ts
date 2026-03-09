import { Component, OnInit, OnDestroy } from '@angular/core';
import { DashboardService } from '../../services/dashboard.service';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  templateUrl: './dashboard.html',
  styleUrls: ['./dashboard.scss'],
  imports: [CommonModule, MatIconModule]
})
export class DashboardComponent implements OnInit, OnDestroy {
  loading = true;
  error = '';
  private refreshInterval: any;

  kpi = {
    transactions: 0,
    anomaliesDetectees: 0,
    anomaliesTraitees: 0,
    anomaliesNonTraitees: 0,
    comptesSurveilles: 0,
    montantSuspect: 0,
    frequenceAnormale: 0,
    entitesBlacklistees: 0,
    paysARisque: 0
  };

  constructor(
    private readonly dashboardService: DashboardService,
    private readonly router: Router
  ) {}

  ngOnInit(): void {
    if (globalThis.window !== undefined) {
      document.title = 'Cockpit LAB';
      this.loadSummary();
      // Rafraîchir les données automatiquement toutes les 30 secondes
      this.refreshInterval = setInterval(() => this.loadSummary(), 30000);
    }
  }

  ngOnDestroy(): void {
    // Nettoyer l'intervalle à la destruction du composant
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }
  }

  loadSummary() {
    this.loading = true;
    this.dashboardService.getDashboardSummary().subscribe({
      next: (data: any) => {
        this.kpi.transactions = data.transactions_count || 0;
        this.kpi.anomaliesDetectees = data.anomalies_count || 0;
        this.kpi.anomaliesTraitees = data.anomalies_treated || 0;
        this.kpi.comptesSurveilles = data.monitored_accounts || 0;

        // Calculer les anomalies non traitées
        this.kpi.anomaliesNonTraitees = this.kpi.anomaliesDetectees - this.kpi.anomaliesTraitees;

        const byType = data.anomalies_by_type || {};
        this.kpi.montantSuspect = byType['Montant'] || 0;
        this.kpi.frequenceAnormale = byType['frequence'] || 0;
        // Combiner tous les types d'entités sous surveillance (Blacklist, PEP, Surveillés, Interdit)
        this.kpi.entitesBlacklistees = (byType['Blacklist'] || 0) + (byType['PEP'] || 0) + (byType['Surveillés'] || 0) + (byType['Interdit'] || 0);
        this.kpi.paysARisque = byType['Pays à risque'] || 0;

        this.loading = false;
      },
      error: () => {
        this.error = 'Erreur de synchronisation.';
        this.loading = false;
      }
    });
  }

  // Rediriger vers la page Anomalies avec filtre "En cours"
  goToAnomaliesQueue() {
    this.router.navigate(['/anomalies'], {
      queryParams: { statut: 'En cours' }
    });
  }
}