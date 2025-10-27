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

  ngOnInit() {
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

  getAnomalyTypeClass(type: string): string {
    return 'type-' + type.toLowerCase().replace(/\s+/g, '-').replace('à', 'a');
  }

  getOverlayClick() {
    this.onClose();
  }
}
