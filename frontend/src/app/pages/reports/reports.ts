import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-reports',
  standalone: true,
  templateUrl: './reports.html',
  styleUrls: ['./reports.scss'],
  imports: [CommonModule, FormsModule]
})
export class ReportsComponent implements OnInit {
  loading = true;
  error = '';
  
  reports: any[] = [
    {
      id: 1,
      name: 'Rapport Mensuel - Octobre 2025',
      type: 'Mensuel',
      date: '2025-10-24',
      anomalies: 127,
      transactions: 5230,
      status: 'Généré'
    },
    {
      id: 2,
      name: 'Rapport Hebdomadaire - Semaine 43',
      type: 'Hebdomadaire',
      date: '2025-10-20',
      anomalies: 42,
      transactions: 1856,
      status: 'Généré'
    },
    {
      id: 3,
      name: 'Rapport Quotidien - 24 Octobre',
      type: 'Quotidien',
      date: '2025-10-24',
      anomalies: 8,
      transactions: 456,
      status: 'Généré'
    },
    {
      id: 4,
      name: 'Rapport Trimestriel - Q3 2025',
      type: 'Trimestriel',
      date: '2025-09-30',
      anomalies: 312,
      transactions: 15670,
      status: 'Généré'
    }
  ];

  filters = {
    type: '',
    status: ''
  };

  constructor() {}

  ngOnInit(): void {
    // Simuler le chargement
    setTimeout(() => {
      this.loading = false;
    }, 500);
  }

  downloadReport(report: any): void {
  }

  viewReport(report: any): void {
  }

  generateReport(): void {
  }

  get filteredReports() {
    return this.reports.filter(r => {
      const typeOk = !this.filters.type || r.type === this.filters.type;
      const statusOk = !this.filters.status || r.status === this.filters.status;
      return typeOk && statusOk;
    });
  }
}
