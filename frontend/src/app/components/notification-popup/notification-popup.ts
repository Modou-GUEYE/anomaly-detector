import { Component, Input, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-notification-popup',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './notification-popup.html',
  styleUrls: ['./notification-popup.scss']
})
export class NotificationPopupComponent implements OnInit {
  @Input() isOpen = false;
  @Input() anomalies: any[] = [];
  @Output() close = new EventEmitter<void>();
  @Output() assign = new EventEmitter<string>();

  displayedAnomalies: any[] = [];
  private readNotifications: Set<string> = new Set();

  ngOnInit() {
    this.loadReadNotifications();
    this.displayedAnomalies = this.anomalies;
  }

  ngOnChanges() {
    this.displayedAnomalies = this.anomalies;
  }

  onClose() {
    this.close.emit();
  }

  onAssign(anomalyId: string) {
    this.assign.emit(anomalyId);
  }

  // Marquer une notification comme lue
  markAsRead(anomalyId: string) {
    this.readNotifications.add(anomalyId);
    this.saveReadNotifications();
  }

  // Vérifier si une notification est lue
  isRead(anomalyId: string): boolean {
    return this.readNotifications.has(anomalyId);
  }

  // Compter les notifications non lues
  getUnreadCount(): number {
    return this.displayedAnomalies.filter(anomaly => !this.isRead(anomaly.id)).length;
  }

  // Sauvegarder les notifications lues dans localStorage
  private saveReadNotifications() {
    const readIds = Array.from(this.readNotifications);
    localStorage.setItem('readNotifications', JSON.stringify(readIds));
  }

  // Charger les notifications lues depuis localStorage
  private loadReadNotifications() {
    const stored = localStorage.getItem('readNotifications');
    if (stored) {
      try {
        const readIds = JSON.parse(stored);
        this.readNotifications = new Set(readIds);
      } catch (e) {
        this.readNotifications = new Set();
      }
    }
  }

  getAnomalyTypeClass(type: string): string {
    return 'type-' + type.toLowerCase().replace(/\s+/g, '-').replace('à', 'a');
  }

  getOverlayClick() {
    this.onClose();
  }
}
