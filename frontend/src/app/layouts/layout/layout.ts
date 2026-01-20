import { Component, OnInit } from '@angular/core';
import { RouterOutlet, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { ProfileService } from '../../services/profile.service';
import { ConfigService } from '../../services/config.service';
import { NotificationPopupComponent } from '../../components/notification-popup/notification-popup';
import { DashboardService } from '../../services/dashboard.service';
import { AuthStateService } from '../../services/auth-state.service';

@Component({
  selector: 'app-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, NotificationPopupComponent],
  templateUrl: './layout.html',
  styleUrls: ['./layout.scss']
})
export class LayoutComponent implements OnInit {
  profile: any = null;
  showNotificationPopup = false;
  unassignedAnomalies: any[] = [];
  kpi = {
    notifications: 0
  };
  sidebarOpen = false;

  constructor(
    private profileService: ProfileService,
    private dashboardService: DashboardService,
    public router: Router,
    private authState: AuthStateService,
    private config: ConfigService
  ) {}

  // Helpers pour templates
  isAdmin(): boolean {
    try {
      const role = (this.profile?.role || '').toLowerCase();
      return role === this.getAdminRoleLower();
    } catch (e) {
      return (this.profile?.role || '').toLowerCase() === this.getAdminRoleLower();
    }
  }

  isAnalyst(): boolean {
    try {
      const role = (this.profile?.role || '').toLowerCase();
      const analystLower = this.getAnalystRoleLower();
      return role === analystLower || role === 'analyste';
    } catch (e) {
      const r = (this.profile?.role || '').toLowerCase();
      return r === this.getAnalystRoleLower() || r === 'analyste';
    }
  }

  // Helper: retourne le nom de rôle admin en minuscules (fallback 'admin')
  private getAdminRoleLower(): string {
    try {
      const r = this.config.getRoleName('admin');
      return (r || 'admin').toLowerCase();
    } catch (e) {
      return 'admin';
    }
  }

  // Helper: retourne le nom de rôle analyst en minuscules (fallback 'analyst')
  private getAnalystRoleLower(): string {
    try {
      const r = this.config.getRoleName('analyst');
      return (r || 'analyst').toLowerCase();
    } catch (e) {
      return 'analyst';
    }
  }

  showTracking(): boolean {
    // show tracking unless the user is an analyst (or french 'analyste')
    return !this.isAnalyst();
  }

  ngOnInit() {
    if (typeof window === 'undefined') {
      return;
    }

    // Charger le profil
    this.profileService.getProfile().subscribe({
      next: (data: any) => {
        this.profile = data;
      },
      error: () => {
      }
    });

    // Charger les notifications
    this.loadNotifications();

  }

  loadNotifications() {
    this.dashboardService.getDashboardSummary().subscribe({
      next: (data: any) => {
        const recentAnomalies = data.recent_anomalies || [];
        this.unassignedAnomalies = recentAnomalies.filter((a: any) => !a.analysteAssigneID);
        this.updateNotificationCount();
      },
      error: () => {
      }
    });
  }

  // Calculer le nombre de notifications non lues
  updateNotificationCount() {
    const readNotifications = this.getReadNotifications();
    const unreadCount = this.unassignedAnomalies.filter(anomaly => 
      !readNotifications.has(anomaly.id)
    ).length;
    this.kpi.notifications = unreadCount;
  }

  // Récupérer les notifications lues depuis localStorage
  private getReadNotifications(): Set<string> {
    const stored = localStorage.getItem('readNotifications');
    if (stored) {
      try {
        const readIds = JSON.parse(stored);
        return new Set(readIds);
      } catch (e) {
        return new Set();
      }
    }
    return new Set();
  }

  toggleNotificationPopup() {
    this.showNotificationPopup = !this.showNotificationPopup;
    // Mettre à jour le compteur quand on ouvre/ferme le popup
    if (!this.showNotificationPopup) {
      this.updateNotificationCount();
    }
  }

  assignAnomaly(anomalyId: string) {
    this.showNotificationPopup = false;
  }

  onLogout() {
    // Use AuthStateService to perform logout logic (clears token, updates signals and navigates)
    try {
      this.authState.logout();
    } catch (e) {
      // Fallback: clear token and navigate to login
      try { localStorage.removeItem('token'); } catch (e) {}
      this.router.navigate(['/login']);
    }
  }

  navigateTo(path: string) {
    this.router.navigate([path]);
    // Fermer le sidebar sur mobile/tablette
    if (window.innerWidth <= 980) {
      this.sidebarOpen = false;
    }
  }

  toggleSidebar() {
    this.sidebarOpen = !this.sidebarOpen;
  }
  closeSidebar() {
    this.sidebarOpen = false;
  }
}
