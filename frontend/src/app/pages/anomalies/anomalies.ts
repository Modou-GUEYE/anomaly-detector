import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AlertService } from '../../services/alert.service';

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
  imports: [CommonModule, FormsModule],
  templateUrl: './anomalies.html',
  styleUrls: ['./anomalies.scss']
})
export class AnomaliesComponent implements OnInit {
  anomalies: Anomaly[] = [];
  filteredAnomalies: Anomaly[] = [];
  loading = true;
  error = '';
  currentPage = 1;
  pageSize = 5;
  totalPages = 1;
  selectedStatus: { [key: string]: string } = {};

  filters = {
    search: '',
    type: '',
    severite: '',
    statut: '',
    dateDebut: '',
    dateFin: ''
  };

  constructor(private alertService: AlertService) {}

  ngOnInit() {
    if (typeof window === 'undefined') {
      return;
    }
    this.loadAnomalies();
  }

  loadAnomalies() {
    this.loading = true;
    this.alertService.getAnomalies().subscribe({
      next: (data: any) => {
        console.log('Données reçues de l\'API:', data);
        this.anomalies = Array.isArray(data) ? data : (data.anomalies || data.data || []);
        
        this.anomalies.forEach(anomaly => {
          this.selectedStatus[anomaly.id] = anomaly.statut;
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

  updateAnomalyStatus(anomaly: Anomaly) {
    const newStatus = this.selectedStatus[anomaly.id];
    if (!newStatus || newStatus === anomaly.statut) {
      return;
    }
    
    this.alertService.updateAnomalyStatus(anomaly.id, newStatus).subscribe({
      next: () => {
        anomaly.statut = newStatus as any;
        anomaly.expanded = false;
        delete this.selectedStatus[anomaly.id];
      },
      error: (err) => {
        this.selectedStatus[anomaly.id] = anomaly.statut;
      }
    });
  }
}
