import { Component, OnInit } from '@angular/core';
import { RouterOutlet, Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { ProfileService } from '../../services/profile.service';
import { NotificationPopupComponent } from '../../components/notification-popup/notification-popup';
import { DashboardService } from '../../services/dashboard.service';

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

  constructor(
    private profileService: ProfileService,
    private dashboardService: DashboardService,
    public router: Router
  ) {}

  ngOnInit() {
    // N'exécuter côté client uniquement
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
        this.kpi.notifications = this.unassignedAnomalies.length;
      },
      error: () => {
      }
    });
  }

  toggleNotificationPopup() {
    this.showNotificationPopup = !this.showNotificationPopup;
  }

  assignAnomaly(anomalyId: string) {
    this.showNotificationPopup = false;
  }

  onLogout() {
    localStorage.removeItem('token');
    this.router.navigate(['/login']);
  }

  navigateTo(path: string) {
    this.router.navigate([path]);
  }
}
