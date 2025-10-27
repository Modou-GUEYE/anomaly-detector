import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DashboardService } from '../../services/dashboard.service';

interface Anomaly {
  id: string;
  transactionId: string;
  code: string;
  type: string;
  statut: 'Traité' | 'En cours' | 'Non assignée' | 'Assignée';
  message: string;
  niveauCriticite: 'Critique' | 'Haute' | 'Moyenne' | 'Basse';
  scoreRisque: number;
  analysteAssigneID?: string | null;
  dateGeneration: string;
  action: 'bloquer' | 'surveiller';
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

  severiteOptions = ['Critique', 'Haute', 'Moyenne', 'Basse'];
  typeOptions = ['Montant', 'frequence', 'Pays à risque', 'Blacklist', 'Interdit', 'PEP', 'Surveillés'];
  statutOptions = ['Traité', 'En cours'];

  constructor(private dashboardService: DashboardService) {}

  ngOnInit() {
    // N'exécuter côté client uniquement
    if (typeof window === 'undefined') {
      return;
    }
    this.loadAnomalies();
  }

  loadAnomalies() {
    this.loading = true;
    // Essaie d'abord l'endpoint spécifique /api/v1/anomalies
    this.dashboardService.getAnomalies().subscribe({
      next: (data: any) => {
        const anomaliesData = Array.isArray(data) ? data : (data.anomalies || data.data || []);
        this.anomalies = anomaliesData.map((a: any) => ({
          id: a.id,
          transactionId: a.transactionId || a.transaction_id,
          code: a.code,
          type: a.type,
          statut: a.statut,
          message: a.message,
          niveauCriticite: a.niveauCriticite || a.severity,
          scoreRisque: a.scoreRisque || a.score_risque,
          analysteAssigneID: a.analysteAssigneID || a.analiste_assigne_id,
          dateGeneration: a.dateGeneration || a.date_generation || new Date().toISOString(),
          action: a.action,
          numeroCompte: a.numeroCompte || a.numero_compte,
          paysRisque: a.paysRisque || a.pays_risque,
          regleDeclencheeID: a.regleDeclencheeID || a.regle_declenchee_id
        }));
        
        // Initialiser selectedStatus avec le statut actuel de chaque anomalie
        this.anomalies.forEach(anomaly => {
          this.selectedStatus[anomaly.id] = anomaly.statut;
        });
        
        this.applyFilters();
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
      }
    });
  }

  getMockAnomalies(): Anomaly[] {
    return [
      {
        id: '7e458829-73b8-4ad1-88f5-5506c3c8f102',
        transactionId: '558921aa-aafb-4eb2-ad0a-e8526aed1ff4',
        code: 'A003',
        type: 'Pays à risque',
        statut: 'Traité',
        message: 'Transaction vers ou depuis un pays à risque.',
        niveauCriticite: 'Haute',
        scoreRisque: 1,
        analysteAssigneID: null,
        dateGeneration: '2025-10-22T12:13:47.963166Z',
        action: 'bloquer',
        numeroCompte: 'SN07900040099887766554406',
        paysRisque: 'Iran',
        regleDeclencheeID: '5fa86fdd-31bf-4698-a538-c3ea3ae84410'
      },
      {
        id: 'e7e56058-999e-49d6-8d52-600f52664fdf',
        transactionId: 'f1b623ff-82fe-4bf8-a634-9bdba9a0e673',
        code: 'A001',
        type: 'Montant',
        statut: 'Traité',
        message: 'Cumul sur 7 jours dépassant le seuil autorisé.',
        niveauCriticite: 'Haute',
        scoreRisque: 1,
        analysteAssigneID: null,
        dateGeneration: '2025-10-22T11:58:55.812085Z',
        action: 'bloquer',
        numeroCompte: '',
        regleDeclencheeID: '9ceaefe9-c9ae-4207-b02b-5f6c1d1db472'
      },
      {
        id: '0cfeac04-e661-429e-8c0f-734368ca0072',
        transactionId: 'db1fd85d-3f7a-4ac7-833e-e45c0ed5d726',
        code: 'A002',
        type: 'frequence',
        statut: 'Traité',
        message: 'Nombre de transactions sur 24h dépassant le seuil autorisé.',
        niveauCriticite: 'Moyenne',
        scoreRisque: 0.7,
        analysteAssigneID: 'f94c55e3-cd37-4c6f-8e8c-ce59e75e178c',
        dateGeneration: '2025-10-22T11:57:35.273192Z',
        action: 'surveiller',
        numeroCompte: 'SN07900040099887766554401',
        regleDeclencheeID: '55b96e1d-1848-42d2-9be8-f53f5791b0a7'
      },
      {
        id: '18bd97d2-c207-4331-a597-2b696ba125db',
        transactionId: 'db1fd85d-3f7a-4ac7-833e-e45c0ed5d726',
        code: 'A002',
        type: 'frequence',
        statut: 'En cours',
        message: 'Nombre de transactions sur 24h dépassant le seuil autorisé.',
        niveauCriticite: 'Moyenne',
        scoreRisque: 0.7,
        analysteAssigneID: null,
        dateGeneration: '2025-10-22T11:57:35.27237Z',
        action: 'surveiller',
        numeroCompte: 'SN07900010012345678901200',
        regleDeclencheeID: '55b96e1d-1848-42d2-9be8-f53f5791b0a7'
      }
    ];
  }

  padNumber(num: number): string {
    return String(num).padStart(2, '0');
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

  assignAnomaly(anomaly: Anomaly) {
    anomaly.statut = 'Assignée';
    anomaly.analysteAssigneID = 'current-user-id';
  }

  resolveAnomaly(anomaly: Anomaly) {
    anomaly.statut = 'Traité';
  }

  rejectAnomaly(anomaly: Anomaly) {
    anomaly.statut = 'En cours';
  }

  getSeveriteClass(severite: string): string {
    switch (severite) {
      case 'Critique':
        return 'severite-critique';
      case 'Haute':
        return 'severite-haute';
      case 'Moyenne':
        return 'severite-moyen';
      case 'Basse':
        return 'severite-bas';
      default:
        return '';
    }
  }

  getStatutClass(statut: string): string {
    switch (statut) {
      case 'Non assignée':
        return 'statut-non-assignee';
      case 'Assignée':
        return 'statut-assignee';
      case 'En cours':
        return 'statut-en-cours';
      case 'Traité':
        return 'statut-traitee';
      default:
        return '';
    }
  }

  correctAllAnomalies() {
    this.anomalies.forEach(a => {
      if (a.statut !== 'Traité') {
        a.statut = 'Traité';
        a.selected = false;
      }
    });
    this.applyFilters();
  }

  markIncomplete(anomaly: Anomaly) {
    anomaly.expanded = false;
  }

  updateAnomalyStatus(anomaly: Anomaly) {
    const newStatus = this.selectedStatus[anomaly.id];
    if (!newStatus || newStatus === anomaly.statut) {
      return;
    }
    
    // Appel API pour persister le changement
    this.dashboardService.updateAnomalyStatus(anomaly.id, newStatus).subscribe({
      next: () => {
        anomaly.statut = newStatus as any;
        anomaly.expanded = false;
        delete this.selectedStatus[anomaly.id];
      },
      error: (err) => {
        // En cas d'erreur, réinitialiser le statut dans le UI
        this.selectedStatus[anomaly.id] = anomaly.statut;
      }
    });
  }
}
