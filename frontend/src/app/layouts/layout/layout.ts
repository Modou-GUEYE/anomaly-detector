import { Component, OnInit } from '@angular/core';
import { RouterOutlet, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { ProfileService } from '../../services/profile.service';
import { ConfigService } from '../../services/config.service';
import { NotificationPopupComponent } from '../../components/notification-popup/notification-popup';
import { DashboardService } from '../../services/dashboard.service';
import { AuthStateService } from '../../services/auth-state.service';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, NotificationPopupComponent, MatIconModule],
  templateUrl: './layout.html',
  styleUrls: ['./layout.scss']
})
export class LayoutComponent implements OnInit {
  profile: any = null;
  showNotificationPopup = false;
  unassignedAnomalies: any[] = [];
  kpi = { notifications: 0 };
  sidebarOpen = false;

  constructor(
    private profileService: ProfileService,
    private dashboardService: DashboardService,
    public router: Router,
    private authState: AuthStateService,
    private config: ConfigService
  ) {}

  // Logique pour obtenir les 2 initiales (John Doe -> JD)
  getInitials(): string {
    const name = this.profile?.nomUtilisateur;
    if (!name) return 'AD'; // Fallback si le nom n'est pas chargé

    const parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      // Prend la première lettre du premier et du deuxième mot
      return (parts[0][0] + parts[1][0]).toUpperCase();
    } else if (parts.length === 1 && parts[0].length > 1) {
      // Si un seul mot, prend les deux premières lettres
      return parts[0].substring(0, 2).toUpperCase();
    }
    return parts[0][0].toUpperCase();
  }

  isAdmin(): boolean {
    const role = (this.profile?.role || '').toLowerCase();
    return role === (this.config.getRoleName('admin') || 'admin').toLowerCase();
  }

  isAnalyst(): boolean {
    const role = (this.profile?.role || '').toLowerCase();
    const analystRole = (this.config.getRoleName('analyst') || 'analyst').toLowerCase();
    return role === analystRole || role === 'analyste';
  }

  showTracking(): boolean {
    return !this.isAnalyst();
  }

  ngOnInit() {
    if (typeof window === 'undefined') return;

    this.profileService.getProfile().subscribe({
      next: (data: any) => this.profile = data,
      error: () => {}
    });

    this.loadNotifications();
  }

  loadNotifications() {
    this.dashboardService.getDashboardSummary().subscribe({
      next: (data: any) => {
        const recentAnomalies = data.recent_anomalies || [];
        this.unassignedAnomalies = recentAnomalies.filter((a: any) => !a.analysteAssigneID);
        this.updateNotificationCount();
      },
      error: () => {}
    });
  }

  updateNotificationCount() {
    const stored = localStorage.getItem('readNotifications');
    let readIds = new Set();
    if (stored) {
      try { readIds = new Set(JSON.parse(stored)); } catch (e) {}
    }
    this.kpi.notifications = this.unassignedAnomalies.filter(a => !readIds.has(a.id)).length;
  }

  toggleNotificationPopup() {
    this.showNotificationPopup = !this.showNotificationPopup;
    if (!this.showNotificationPopup) this.updateNotificationCount();
  }

  assignAnomaly(anomalyId: string) {
    this.showNotificationPopup = false;
  }

  onLogout() {
    try {
      this.authState.logout();
    } catch (e) {
      localStorage.removeItem('token');
      this.router.navigate(['/login']);
    }
  }

  navigateTo(path: string) {
    this.router.navigate([path]);
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