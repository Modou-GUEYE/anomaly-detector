import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NgChartsModule } from 'ng2-charts';
import { ChartOptions, ChartData } from 'chart.js';

@Component({
  selector: 'app-anomaly-chart',
  standalone: true,
  imports: [CommonModule, NgChartsModule],
  templateUrl: './anomaly-chart.html',
  styleUrls: ['./anomaly-chart.scss']
})
export class AnomalyChartComponent {
  @Input() mode: string = 'overview';
  // Données statiques pour garantir l'affichage des 4 anomalies

  public barChartOptions: ChartOptions<'bar'> = {
    responsive: true,
    plugins: {
      legend: { 
        display: true,
        position: 'top',
        labels: {
          color: '#24288C',
          font: { size: 12, weight: 'bold' }
        }
      },
      title: { display: false }
    },
    scales: {
      y: {
        beginAtZero: true,
        ticks: { color: '#475569' },
        grid: { color: '#E3E3E1' }
      },
      x: {
        ticks: { color: '#475569' },
        grid: { display: false }
      }
    }
  };
  public barChartData: ChartData<'bar'> = {
    labels: ['Jan', 'Fév', 'Mar', 'Avr', 'Mai', 'Jun', 'Jul', 'Aoû', 'Sep', 'Oct', 'Nov', 'Déc'],
    datasets: [
      {
        label: 'PaysARisque',
        data: [40, 35, 30, 45, 50, 38, 42, 47, 44, 41, 39, 48],
        backgroundColor: '#6D5AE0',
        borderRadius: 6,
        hidden: false
      },
      {
        label: 'Montant Suspect',
        data: [12, 15, 18, 22, 19, 25, 28, 24, 30, 27, 32, 35],
        backgroundColor: '#2463DE',
        borderRadius: 6
      },
      {
        label: 'Fréquence Anormale',
        data: [8, 10, 12, 15, 14, 18, 20, 17, 22, 19, 24, 26],
        backgroundColor: '#ED6A01',
        borderRadius: 6
      },
      {
        label: 'Entités Blacklistées',
        data: [3, 4, 2, 5, 6, 4, 7, 5, 8, 6, 9, 7],
        backgroundColor: '#EF4444',
        borderRadius: 6
      }
    ]
  };
}
