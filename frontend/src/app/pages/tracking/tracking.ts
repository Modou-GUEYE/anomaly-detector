import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { ConfigService } from '../../services/config.service';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Subject, Observable, of } from 'rxjs';
import { takeUntil, debounceTime, distinctUntilChanged, catchError } from 'rxjs/operators';

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
  search: string;
}

@Component({
  selector: 'app-tracking',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './tracking.html',
  styleUrls: ['./tracking.scss']
})
export class TrackingComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();
  public searchSubject = new Subject<string>(); // Rendu public pour le template

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
    search: ''
  };

  currentPage = 1;
  pageSize = 20;
  totalPages = 1;
  paginatedEvents: any[] = [];

  severityOptions = ['succès', 'bloqué', 'échec', 'surveillance'];
  moduleOptions = ['Alertes', 'Blacklist', 'Règles de Contrôle', 'Pays à Risque', 'Comptes'];

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
        this.error = 'Impossible de charger les données: ' + (error.message || error);
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
        const match = [event.nom_utilisateur, event.action_effectuee, event.resultat, event.type_action]
          .some(f => f?.toString().toLowerCase().includes(s));
        if (!match) return false;
      }
      if (this.filters.severity && event.resultat !== this.filters.severity) return false;
      if (this.filters.module && event.type_action !== this.filters.module) return false;
      return true;
    });
    this.updatePagination();
  }

  updatePagination() {
    this.totalPages = Math.ceil(this.filteredEvents.length / this.pageSize);
    this.currentPage = Math.max(1, Math.min(this.currentPage, this.totalPages));
    const start = (this.currentPage - 1) * this.pageSize;
    this.paginatedEvents = this.filteredEvents.slice(start, start + this.pageSize);
  }

  nextPage() { if (this.currentPage < this.totalPages) { this.currentPage++; this.updatePagination(); } }
  previousPage() { if (this.currentPage > 1) { this.currentPage--; this.updatePagination(); } }
  refreshData() { this.loadRealData(); }
  selectEvent(event: any) { this.selectedEvent = event; }
  closeEventDetails() { this.selectedEvent = null; }

  exportData(format: string) {
    // Logique d'export déjà présente dans ton code initial...
    console.log('Exporting as', format);
  }

  trackByEventId(index: number, event: any) { return event.id || index; }
  getSeverityClass(s: string) { return `severity-${(s || '').toLowerCase()}`; }
  getComplianceClass(s: string) { return `compliance-${(s || 'unknown').toLowerCase()}`; }
}