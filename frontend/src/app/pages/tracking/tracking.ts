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
  private searchSubject = new Subject<string>();

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

  severityOptions = [
    'succès',
    'bloqué',
    'échec',
    'surveillance'
  ];
  
  moduleOptions = [
    'Alertes',
    'Blacklist',
    'Règles de Contrôle',
    'Pays à Risque',
    'Comptes'
  ];

  private apiUrl = '';

  private http = inject(HttpClient);

  // inject ConfigService via constructor for runtime role keywords
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
        console.error('❌ Erreur lors du chargement des données d\'audit:', error);
        this.error = 'Impossible de charger les données d\'audit: ' + (error.message || error);
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
      },
      error: (error) => {
        console.error('💥 Erreur finale lors du chargement:', error);
        this.error = 'Erreur lors du chargement des données d\'audit';
        this.loading = false;
      }
    });
  }

  private getAuditData(): Observable<any[]> {
    let params = new HttpParams();
    if (this.filters.dateStart) params = params.set('date_start', this.filters.dateStart);
    if (this.filters.dateEnd) params = params.set('date_end', this.filters.dateEnd);
    
    return this.http.get<any[]>(`${this.apiUrl}/api/v1/reports/audit`, { params }).pipe(
      catchError(err => {
        console.error('🚨 Erreur API audit:', err);
        return of([]);
      })
    );
  }

  calculateStats() {
    const total = this.events.length;
    
    // Événements critiques: alertes avec resultat "bloqué", "échec", "erreur"
    const criticalEvents = this.events.filter(e => {
      const details = (e.details || '').toLowerCase();
      const action = (e.action_effectuee || e.action || '').toLowerCase();
      const resultat = (e.resultat || '').toLowerCase();
      return resultat === 'bloqué' || 
             resultat === 'échec' || 
             resultat === 'erreur' ||
             details.includes('bloquer') || 
             action.includes('bloquer');
    }).length;
    
    // Utilisateurs actifs: compter les utilisateur_id ou role uniques
    const activeUsers = new Set(
      this.events.map(e => e.utilisateur_id || e.user_id || e.role || 'unknown')
    ).size;
    
    // Taux de conformité: pourcentage d'événements avec resultat "succès"
    const successEvents = this.events.filter(e => 
      (e.resultat || '').toLowerCase() === 'succès'
    ).length;
    const complianceRate = total > 0 ? Math.round((successEvents / total) * 100) : 0;
    
    // Événements aujourd'hui
    const today = new Date().toDateString();
    const eventsToday = this.events.filter(e => {
      const eventDate = new Date(e.date_heure || e.timestamp || e.created_at || Date.now());
      return eventDate.toDateString() === today;
    }).length;
    
    this.stats = {
      totalEvents: total,
      criticalEvents: criticalEvents,
      activeUsers: activeUsers,
      complianceRate: complianceRate,
      eventsToday: eventsToday
    };
  }

  applyFilters() {
    this.filteredEvents = this.events.filter(event => {
      if (this.filters.search) {
        const searchTerm = this.filters.search.trim().toLowerCase();
        const userFields = [
          event.nom_utilisateur,
          event.utilisateur_id,
          event.role || event.username
        ].filter(Boolean) as string[];
        const globalFields = [
          ...userFields,
          event.action_effectuee || event.action,
          event.details,
          event.type_action,
          event.resultat
        ].filter(Boolean) as string[];

        const matches = (fields: string[]) =>
          fields.some(field => field.toString().toLowerCase().includes(searchTerm));

        // build role keywords dynamically from runtime config, keep common aliases
        const roleKeywords = (() => {
          const out: string[] = [];
          try {
            const admin = this.config.getRoleName('admin'); if (admin) out.push(admin.toLowerCase());
            const analyst = this.config.getRoleName('analyst'); if (analyst) out.push(analyst.toLowerCase());
            const system = this.config.getRoleName('system'); if (system) out.push(system.toLowerCase());
          } catch (e) {
            // push configured role names (fallback to common literals)
            try {
              const adminRole = this.config.getRoleName('admin'); if (adminRole) out.push(adminRole.toLowerCase());
            } catch (e) { out.push('admin'); }
            try {
              const analystRole = this.config.getRoleName('analyst'); if (analystRole) out.push(analystRole.toLowerCase());
            } catch (e) { out.push('analyst'); }
            try {
              const systemRole = this.config.getRoleName('system'); if (systemRole) out.push(systemRole.toLowerCase());
            } catch (e) { out.push('system'); }
          }
          // common alternative names
          // garder quelques alias usuels (français/identifiants d'application)
          out.push('analyste', 'application', 'app', 'mda');
          return Array.from(new Set(out));
        })();
        const shouldRestrictToUser = roleKeywords.includes(searchTerm);

        const searchMatch = shouldRestrictToUser ? matches(userFields) : matches(globalFields);
        if (!searchMatch) return false;
      }

      // Filtre par Résultat (colonne resultat: succès, bloqué, échec, etc.)
      if (this.filters.severity) {
        const resultat = event.resultat;
        if (resultat !== this.filters.severity) {
          return false;
        }
      }

      // Filtre par Module (colonne type_action: Alertes, Blacklist, etc.)
      if (this.filters.module) {
        const typeAction = event.type_action;
        if (typeAction !== this.filters.module) {
          return false;
        }
      }

      return true;
    });

    this.updatePagination();
  }

  updatePagination() {
    this.totalPages = Math.ceil(this.filteredEvents.length / this.pageSize);
    this.currentPage = Math.min(this.currentPage, Math.max(1, this.totalPages));
    
    const startIndex = (this.currentPage - 1) * this.pageSize;
    const endIndex = startIndex + this.pageSize;
    this.paginatedEvents = this.filteredEvents.slice(startIndex, endIndex);
  }

  nextPage() {
    if (this.currentPage < this.totalPages) {
      this.currentPage++;
      this.updatePagination();
    }
  }

  previousPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
      this.updatePagination();
    }
  }

  refreshData() {
    this.loadRealData();
  }

  selectEvent(event: any) {
    this.selectedEvent = event;
  }

  closeEventDetails() {
    this.selectedEvent = null;
  }

  exportData(format: 'csv' | 'excel') {
    let params = new HttpParams();
    
    if (this.filters.module) {
      params = params.set('module', this.filters.module);
    }
    if (this.filters.severity) {
      params = params.set('statut', this.filters.severity);
    }
    if (this.filters.search) {
      params = params.set('action', this.filters.search);
    }
    if (this.filters.dateStart) {
      params = params.set('date_debut', this.filters.dateStart);
    }
    if (this.filters.dateEnd) {
      params = params.set('date_fin', this.filters.dateEnd);
    }

    this.http.get('/api/v1/reports/audit/export', {
      params: params,
      responseType: 'blob',
      observe: 'response'
    }).subscribe({
      next: (response) => {
        const blob = response.body;
        if (blob) {
          const url = window.URL.createObjectURL(blob);
          const link = document.createElement('a');
          link.href = url;
          
          const contentDisposition = response.headers.get('content-disposition');
          let filename = 'audit_export.xlsx';
          if (contentDisposition) {
            const matches = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/.exec(contentDisposition);
            if (matches != null && matches[1]) {
              filename = matches[1].replace(/['"]/g, '');
            }
          }
          
          link.download = filename;
          document.body.appendChild(link);
          link.click();
          document.body.removeChild(link);
          window.URL.revokeObjectURL(url);
        }
      },
      error: (error) => {
        console.error('Erreur lors de l\'export:', error);
        this.error = 'Erreur lors de l\'export des données';
      }
    });
  }

  trackByEventId(index: number, event: any): string {
    return event.id || event._id || index.toString();
  }

  getSeverityClass(severity: string): string {
    if (!severity) {
      return 'severity-unknown';
    }
    return `severity-${severity.toString().toLowerCase()}`;
  }

  getComplianceClass(status: string): string {
    if (!status) {
      return 'compliance-unknown';
    }
    return `compliance-${status.toString().toLowerCase().replace('_', '-')}`;
  }

  getFormattedDate(date: Date): string {
    return new Date(date).toLocaleDateString('fr-FR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }
}