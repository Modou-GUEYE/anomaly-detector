import { Component, OnInit, OnDestroy, inject } from '@angular/core';
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
  avgResponseTime: number;
  eventsToday: number;
}

interface TrackingFilters {
  dateStart: string;
  dateEnd: string;
  eventType: string;
  severity: string;
  userId: string;
  module: string;
  compliance: string;
  search: string;
  realTime: boolean;
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
    avgResponseTime: 0,
    eventsToday: 0
  };

  loading = true;
  error = '';
  selectedEvent: any = null;
  
  // ...existing code...
  filters: TrackingFilters = {
    dateStart: '',
    dateEnd: '',
    eventType: '',
    severity: '',
    userId: '',
    module: '',
    compliance: '',
    search: '',
    realTime: true
  };

  // ...existing code...
  currentPage = 1;
  pageSize = 20;
  totalPages = 1;
  paginatedEvents: any[] = [];

  // ...existing code...
  eventTypeOptions = [
    'Alertes',
    'Blacklist', 
    'Règles de Contrôle',
    'Pays à Risque',
    'Comptes'
  ];
  
  severityOptions = [
    'succès',
    'bloqué',
    'échec',
    'erreur',
    'surveillance',
    'notifié'
  ];
  
  moduleOptions = [
    'Alertes',
    'Blacklist',
    'Règles de Contrôle',
    'Pays à Risque',
    'Comptes'
  ];

  // ...existing code...
  private apiUrl = '';

  // ...existing code...
  private http = inject(HttpClient);

  constructor() {}

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
      avgResponseTime: 125,
      eventsToday: eventsToday
    };
  }

  applyFilters() {
    this.filteredEvents = this.events.filter(event => {
      if (this.filters.search) {
        const searchTerm = this.filters.search.toLowerCase();
        const searchFields = [
          event.role || event.username,
          event.action_effectuee || event.action,
          event.details,
          event.type_action,
          event.resultat
        ].filter(field => field);
        
        const searchMatch = searchFields.some(field => 
          field.toString().toLowerCase().includes(searchTerm)
        );
        
        if (!searchMatch) return false;
      }

      // Filtre par Type d'Action (colonnes type_action)
      if (this.filters.eventType) {
        const typeAction = event.type_action;
        if (typeAction !== this.filters.eventType) {
          return false;
        }
      }

      // Filtre par Résultat (colonne resultat: succès, bloqué, échec, etc.)
      if (this.filters.severity) {
        const resultat = event.resultat;
        if (resultat !== this.filters.severity) {
          return false;
        }
      }

      // Filtre par Module (même que type_action)
      if (this.filters.module) {
        const module = event.type_action;
        if (module !== this.filters.module) {
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
    const data = this.formatDataForExport(this.filteredEvents, format);
    const blob = new Blob([data], { 
      type: format === 'csv' ? 'text/csv' : 'application/octet-stream' 
    });
    
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `tracking-events.${format}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }

  private formatDataForExport(events: any[], format: string): string {
    if (format === 'csv') {
      const headers = 'Date,Utilisateur,Événement,Description,Détails\n';
      const rows = events.map(event => {
        const date = event.timestamp || event.created_at || new Date();
        const user = event.userName || event.user_name || event.username || 'N/A';
        const eventType = event.eventType || event.event_type || event.action || 'N/A';
        const description = event.description || event.message || event.details || 'N/A';
        const severity = event.severity || event.level || event.priority || 'N/A';
        
        return `"${new Date(date).toLocaleString()}","${user}","${eventType}","${description}","${severity}"`;
      }).join('\n');
      return headers + rows;
    }
    return JSON.stringify(events, null, 2);
  }

  // ...existing code...
  trackByEventId(index: number, event: any): string {
    return event.id || event._id || index.toString();
  }

  getEventTypeIcon(eventType: string): string {
    const icons = {
      'LOGIN': '🔐',
      'LOGOUT': '🚪',
      'ANOMALY_DETECTED': '🚨',
      'ANOMALY_RESOLVED': '✅',
      'RULE_TRIGGERED': '⚡',
      'ACCOUNT_BLOCKED': '🔒',
      'SUSPICIOUS_ACTIVITY': '⚠️',
      'COMPLIANCE_CHECK': '📋',
      'DATA_EXPORT': '📤',
      'USER_ACTION': '👤'
    };
    return icons[eventType as keyof typeof icons] || '📝';
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

  getEventSource(userName: string): string {
    if (!userName) {
      return 'Source inconnue';
    }
    
    if (userName === 'MDA') {
      return 'Moteur de Détection Automatique';
    } else if (userName === 'admin') {
      return 'Administrateur Système';
    } else if (userName === 'analyst') {
      return 'Analyste Utilisateur';
    } else if (userName === 'system') {
      return 'Système Automatisé';
    } else {
      return `Utilisateur: ${userName}`;
    }
  }

  getObjectKeys(obj: any): string {
    return Object.keys(obj || {}).join(', ');
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