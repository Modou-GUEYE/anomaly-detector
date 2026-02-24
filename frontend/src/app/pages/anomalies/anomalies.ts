import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { AlertService } from '../../services/alert.service';
import { MatIconModule } from '@angular/material/icon';

interface Anomaly {
  id: string;
  transactionId: string;
  code: string;
  type: string;
  statut: string;
  message: string;
  niveauCriticite: string;
  scoreRisque: number;
  analysteAssigneID?: string | null;
  analysteNom?: string;
  dateGeneration: string;
  action: string;
  numeroCompte: string;
  paysRisque?: string;
  regleDeclencheeID?: string;
  selected?: boolean;
  expanded?: boolean;
}

@Component({
  selector: 'app-anomalies',
  standalone: true,
  imports: [CommonModule, FormsModule, MatIconModule],
  templateUrl: './anomalies.html',
  styleUrls: ['./anomalies.scss']
})
export class AnomaliesComponent implements OnInit {
  anomalies: Anomaly[] = [];
  filteredAnomalies: Anomaly[] = [];
  severityOptions: string[] = [];
  loading = true;
  error = '';
  currentPage = 1;
  pageSize = 10;
  totalPages = 1;
  selectedStatus: { [key: string]: string } = {};
  showDetailsModal = false;
  detailsAnomaly: Anomaly | null = null;

  filters = {
    search: '',
    type: '',
    severite: '',
    statut: '',
    dateDebut: '',
    dateFin: ''
  };

  constructor(private alertService: AlertService, private route: ActivatedRoute) {}

  ngOnInit() {
    if (typeof window === 'undefined') {
      return;
    }
    // Lire les query parameters depuis l'URL (ex: ?statut=En cours)
    this.route.queryParams.subscribe(params => {
      if (params['statut']) {
        this.filters.statut = params['statut'];
      }
    });
    this.loadAnomalies();
  }

  loadAnomalies() {
    this.loading = true;
    this.alertService.getAnomalies().subscribe({
      next: (data: any) => {
        this.anomalies = Array.isArray(data) ? data : (data.anomalies || data.data || []);
        
          // Normaliser les libellés de sévérité et préparer les options de filtre
          this.anomalies.forEach(anomaly => {
            this.selectedStatus[anomaly.id] = anomaly.statut;
            anomaly.niveauCriticite = this.normalizeSeverity(anomaly.niveauCriticite);
          });

          // Construire la liste unique d'options de sévérité dans un ordre raisonnable
          const order = ['Critique', 'Haute', 'Moyenne', 'Faible'];
          const uniques = Array.from(new Set(this.anomalies.map(a => a.niveauCriticite).filter(Boolean)));
          this.severityOptions = uniques.sort((a, b) => {
            const ia = order.indexOf(a) === -1 ? order.length : order.indexOf(a);
            const ib = order.indexOf(b) === -1 ? order.length : order.indexOf(b);
            return ia - ib || a.localeCompare(b);
          });
        
        this.applyFilters();
        this.loading = false;
      },
      error: (err) => {
        console.error('Erreur lors du chargement des anomalies:', err);
        this.error = 'Impossible de charger les anomalies';
        this.loading = false;
      }
    });
  }

  /**
   * Normalise différents libellés/valeurs de sévérité en labels FR cohérents
   */
  normalizeSeverity(value?: string): string {
    if (!value) return '';
    const v = value.toString().trim().toLowerCase();
    if (!v) return '';

    if (['critique', 'critical', 'severe'].includes(v)) return 'Critique';
    if (['haute', 'high'].includes(v)) return 'Haute';
    if (['moyenne', 'medium'].includes(v)) return 'Moyenne';
    if (['basse', 'faible', 'low'].includes(v)) return 'Faible';

    // Capitaliser le premier caractère par défaut
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  applyFilters() {
    let result = [...this.anomalies];

    // Recherche textuelle
    if (this.filters.search) {
      const search = this.filters.search.toLowerCase();
      result = result.filter(a =>
        a.type.toLowerCase().includes(search) ||
        a.code.toLowerCase().includes(search)
      );
    }

    // Filtres
    if (this.filters.type) {
      result = result.filter(a => a.type === this.filters.type);
    }
    if (this.filters.severite) {
      result = result.filter(a => a.niveauCriticite === this.filters.severite);
    }
    if (this.filters.statut) {
      result = result.filter(a => a.statut === this.filters.statut);
    }

    this.totalPages = Math.ceil(result.length / this.pageSize);
    this.currentPage = 1;
    this.filteredAnomalies = result;
  }

  get paginatedAnomalies() {
    const start = (this.currentPage - 1) * this.pageSize;
    const end = start + this.pageSize;
    return this.filteredAnomalies.slice(start, end);
  }

  nextPage() {
    if (this.currentPage < this.totalPages) {
      this.currentPage++;
    }
  }

  prevPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
    }
  }

  openDetailsModal(anomaly: Anomaly) {
    this.detailsAnomaly = anomaly;
    this.showDetailsModal = true;
  }

  closeDetailsModal() {
    this.showDetailsModal = false;
    this.detailsAnomaly = null;
  }

  updateAnomalyStatus(anomaly: Anomaly) {
    const newStatus = this.selectedStatus[anomaly.id];
    if (!newStatus || newStatus === anomaly.statut) {
      return;
    }
    
    this.alertService.updateAnomalyStatus(anomaly.id, newStatus).subscribe({
      next: () => {
        anomaly.statut = newStatus as any;
        this.closeDetailsModal();
        delete this.selectedStatus[anomaly.id];
      },
      error: (err) => {
        this.selectedStatus[anomaly.id] = anomaly.statut;
      }
    });
  }

  getAnomalyStatusClass(status: string): string {
    const normalized = status?.toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/\s+/g, '');
    switch (normalized) {
      case 'traite':
        return 'status-trait';
      case 'encours':
        return 'status-encours';
      default:
        return 'status-unknown';
    }
  }
}
