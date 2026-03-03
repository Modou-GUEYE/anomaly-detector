import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { ConfigService } from '../../services/config.service';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient, HttpParams, HttpErrorResponse } from '@angular/common/http';
import { Subject, Observable, of } from 'rxjs';
import { takeUntil, debounceTime, distinctUntilChanged, catchError } from 'rxjs/operators';
import * as XLSX from 'xlsx';
import { MatIconModule } from '@angular/material/icon';

interface TrackingStats {
  totalEvents: number;
  activeUsers: number;
  criticalEvents: number;
  complianceRate: number;
  eventsToday: number;
}

interface TrackingFilters {
  dateStart: string;
  dateEnd: string;
  severity: string;
  module: string;
  user: string;
  search: string;
}

@Component({
  selector: 'app-tracking',
  standalone: true,
  imports: [CommonModule, FormsModule, MatIconModule],
  templateUrl: './tracking.html',
  styleUrls: ['./tracking.scss']
})
export class TrackingComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();
  public searchSubject = new Subject<string>(); 

  events: any[] = [];
  filteredEvents: any[] = [];
  stats: TrackingStats = {
    totalEvents: 0,
    activeUsers: 0,
    criticalEvents: 0,
    complianceRate: 0,
    eventsToday: 0
  };

  loading = true;
  error = '';
  selectedEvent: any = null;
  
  filters: TrackingFilters = {
    dateStart: '',
    dateEnd: '',
    severity: '',
    module: '',
    user: '',
    search: ''
  };

  currentPage = 1;
  pageSize = 20;
  totalPages = 1;
  paginatedEvents: any[] = [];

  severityOptions = ['succès', 'bloqué', 'échec', 'surveillance'];
  moduleOptions = ['Alertes', 'Blacklist', 'Règles de Contrôle', 'Pays à Risque', 'Comptes'];
  userOptions: string[] = [];

  private apiUrl = '';
  private http = inject(HttpClient);

  constructor(private config: ConfigService) {}

  ngOnInit() {
    this.initializeFilters();
    this.setupSearchSubscription();
    this.loadRealData();
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  initializeFilters() {
    const today = new Date();
    const thirtyDaysAgo = new Date(today.getTime() - 30 * 24 * 60 * 60 * 1000);
    this.filters.dateStart = thirtyDaysAgo.toISOString().split('T')[0];
    this.filters.dateEnd = today.toISOString().split('T')[0];
  }

  setupSearchSubscription() {
    this.searchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      takeUntil(this.destroy$)
    ).subscribe(searchTerm => {
      this.filters.search = searchTerm;
      this.applyFilters();
    });
  }

  loadRealData() {
    this.loading = true;
    this.error = '';
    this.getAuditData().pipe(
      takeUntil(this.destroy$),
      catchError(error => {
        this.error = this.getUserFriendlyErrorMessage(error);
        this.loading = false;
        return of([]);
      })
    ).subscribe({
      next: (auditData) => {
        this.events = auditData.map((item: any) => ({
          ...item,
          id: item.id || Math.random().toString(36).substr(2, 9),
          timestamp: new Date(item.date_heure || item.created_at || Date.now())
        }));
        this.userOptions = this.buildUserOptions(this.events);
        this.calculateStats();
        this.applyFilters();
        this.loading = false;
      }
    });
  }

  private getAuditData(): Observable<any[]> {
    let params = new HttpParams();
    if (this.filters.dateStart) params = params.set('date_start', this.filters.dateStart);
    if (this.filters.dateEnd) params = params.set('date_end', this.filters.dateEnd);
    return this.http.get<any[]>(`${this.apiUrl}/api/v1/reports/audit`, { params });
  }

  private getUserFriendlyErrorMessage(error: unknown): string {
    const defaultMessage = 'Impossible de charger les données pour le moment. Veuillez réessayer.';

    if (error instanceof HttpErrorResponse) {
      if (error.status === 0) {
        return 'Le serveur backend est indisponible ou non démarré. Vérifiez que l’API est lancée puis rechargez la page.';
      }

      if (error.status >= 500) {
        return 'Erreur côté serveur. Veuillez réessayer plus tard.';
      }

      if (error.status >= 400) {
        return 'La requête de chargement des données a échoué. Veuillez vérifier votre session puis réessayer.';
      }
    }

    return defaultMessage;
  }

  calculateStats() {
    const total = this.events.length;
    const criticalEvents = this.events.filter(e => {
      const res = (e.resultat || '').toLowerCase();
      return res === 'bloqué' || res === 'échec' || res === 'erreur';
    }).length;
    
    const activeUsers = new Set(this.events.map(e => e.utilisateur_id || e.role || 'unknown')).size;
    const successEvents = this.events.filter(e => (e.resultat || '').toLowerCase() === 'succès').length;
    
    this.stats = {
      totalEvents: total,
      criticalEvents: criticalEvents,
      activeUsers: activeUsers,
      complianceRate: total > 0 ? Math.round((successEvents / total) * 100) : 0,
      eventsToday: this.events.filter(e => new Date(e.timestamp).toDateString() === new Date().toDateString()).length
    };
  }

  applyFilters() {
    this.filteredEvents = this.events.filter(event => {
      if (this.filters.search) {
        const s = this.filters.search.toLowerCase();
        const match = [event.action_effectuee, this.getEventUser(event)]
          .some(f => f?.toString().toLowerCase().includes(s));
        if (!match) return false;
      }
      if (this.filters.severity && event.resultat !== this.filters.severity) return false;
      if (this.filters.module && event.type_action !== this.filters.module) return false;
      if (this.filters.user) {
        const eventUser = this.getEventUser(event);
        if (eventUser !== this.filters.user) return false;
      }
      return true;
    });
    this.updatePagination();
  }

  private getEventUser(event: any): string {
    return (event.nom_utilisateur || event.userName || '').toString().trim();
  }

  private buildUserOptions(events: any[]): string[] {
    return Array.from(new Set(events
      .map(e => this.getEventUser(e))
      .filter(name => !!name)))
      .sort((a, b) => a.localeCompare(b, 'fr', { sensitivity: 'base' }));
  }

  updatePagination() {
    this.totalPages = Math.ceil(this.filteredEvents.length / this.pageSize);
    this.currentPage = Math.max(1, Math.min(this.currentPage, this.totalPages));
    const start = (this.currentPage - 1) * this.pageSize;
    this.paginatedEvents = this.filteredEvents.slice(start, start + this.pageSize);
  }

  nextPage() { if (this.currentPage < this.totalPages) { this.currentPage++; this.updatePagination(); } }
  previousPage() { if (this.currentPage > 1) { this.currentPage--; this.updatePagination(); } }
  selectEvent(event: any) { this.selectedEvent = event; }
  closeEventDetails() { this.selectedEvent = null; }

  exportData(format: string) {
    if (!this.filteredEvents || this.filteredEvents.length === 0) {
      alert('Aucune donnée à exporter. Veuillez charger les données d\'audit.');
      return;
    }

    if (format === 'excel') {
      this.exportToExcel();
    } else if (format === 'csv') {
      this.exportToCSV();
    }
  }

  private exportToExcel() {
    try {
      // Préparation des données au format tableau
      const dataToExport = this.filteredEvents.map(event => ({
        'Horodatage': this.formatDate(event['date_heure'] || event['timestamp']),
        'Heure': this.formatTime(event['date_heure'] || event['timestamp']),
        'Action / Type': event['type_action'] || 'N/A',
        'Utilisateur': event['nom_utilisateur'] || event['userName'] || 'N/A',
        'Rôle': event['role'] || 'N/A',
        'Événement': event['action_effectuee'] || event['eventType'] || 'N/A',
        'Résultat': event['resultat'] || 'N/A',
        'ID Log': event['id'] || 'N/A',
        'Entity ID': event['entity_id'] || 'N/A'
      }));

      // Création du workbook et worksheet
      const worksheet = XLSX.utils.json_to_sheet(dataToExport, {
        header: ['Horodatage', 'Heure', 'Action / Type', 'Utilisateur', 'Rôle', 'Événement', 'Résultat', 'ID Log', 'Entity ID']
      });

      // Ajustement des largeurs de colonnes
      const columnWidths = [
        { wch: 12 },  // Horodatage
        { wch: 10 },  // Heure
        { wch: 20 },  // Action / Type
        { wch: 20 },  // Utilisateur
        { wch: 15 },  // Rôle
        { wch: 30 },  // Événement
        { wch: 12 },  // Résultat
        { wch: 15 },  // ID Log
        { wch: 15 }   // Entity ID
      ];
      worksheet['!cols'] = columnWidths;

      // Création du workbook
      const workbook = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(workbook, worksheet, 'Audit');

      // Génération du nom du fichier avec la date
      const now = new Date();
      const fileName = `audit_export_${now.toISOString().split('T')[0]}_${now.getTime()}.xlsx`;

      // Téléchargement du fichier
      XLSX.writeFile(workbook, fileName);
    } catch (error) {
      console.error('Erreur lors de l\'export Excel:', error);
      alert('Une erreur est survenue lors de l\'export. Veuillez vérifier la console.');
    }
  }

  private exportToCSV() {
    try {
      // Préparation des données au format CSV
      const headers = ['Horodatage', 'Heure', 'Action / Type', 'Utilisateur', 'Rôle', 'Événement', 'Résultat', 'ID Log', 'Entity ID'];
      
      const rows = this.filteredEvents.map(event => [
        this.formatDate(event['date_heure'] || event['timestamp']),
        this.formatTime(event['date_heure'] || event['timestamp']),
        this.escapeCsvField(event['type_action'] || 'N/A'),
        this.escapeCsvField(event['nom_utilisateur'] || event['userName'] || 'N/A'),
        this.escapeCsvField(event['role'] || 'N/A'),
        this.escapeCsvField(event['action_effectuee'] || event['eventType'] || 'N/A'),
        this.escapeCsvField(event['resultat'] || 'N/A'),
        event['id'] || 'N/A',
        event['entity_id'] || 'N/A'
      ]);

      // Création du contenu CSV
      const csvContent = [
        headers.join(','),
        ...rows.map(row => row.join(','))
      ].join('\n');

      // Création du blob
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      
      // Génération du nom du fichier
      const now = new Date();
      const fileName = `audit_export_${now.toISOString().split('T')[0]}_${now.getTime()}.csv`;

      // Téléchargement du fichier
      this.downloadFile(blob, fileName);
    } catch (error) {
      console.error('Erreur lors de l\'export CSV:', error);
      alert('Une erreur est survenue lors de l\'export. Veuillez vérifier la console.');
    }
  }

  private formatDate(date: any): string {
    if (!date) return 'N/A';
    const d = new Date(date);
    return d.toLocaleDateString('fr-FR');
  }

  private formatTime(date: any): string {
    if (!date) return 'N/A';
    const d = new Date(date);
    return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
  }

  private escapeCsvField(field: string): string {
    // Échappe les guillemets et ajoute des guillemets si le champ contient une virgule, un retour à la ligne ou un guillemet
    if (field.includes(',') || field.includes('\n') || field.includes('"')) {
      return `"${field.replace(/"/g, '""')}"`;
    }
    return field;
  }

  private downloadFile(blob: Blob, fileName: string) {
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }

  trackByEventId(index: number, event: any) { return event.id || index; }
  getSeverityClass(s: string) { return `severity-${(s || '').toLowerCase()}`; }
  getComplianceClass(s: string) { return `compliance-${(s || 'unknown').toLowerCase()}`; }
}