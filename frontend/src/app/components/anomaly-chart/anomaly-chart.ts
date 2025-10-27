import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { CommonModule } from '@angular/common';
import { HttpClientModule } from '@angular/common/http';
import { NgChartsModule, BaseChartDirective } from 'ng2-charts';
import { ChartOptions, ChartData } from 'chart.js';

@Component({
  selector: 'app-anomaly-chart',
  standalone: true,
  imports: [CommonModule, NgChartsModule, HttpClientModule],
  templateUrl: './anomaly-chart.html',
  styleUrls: ['./anomaly-chart.scss']
})
export class AnomalyChartComponent implements OnInit {
  @Input() mode: string = 'overview';
  @ViewChild(BaseChartDirective) chart?: BaseChartDirective;

  constructor(private http: HttpClient) {}

  public barChartOptions: ChartOptions<'bar'> = {
    responsive: true,
    interaction: {
      intersect: false,
      mode: 'index'
    },
    plugins: {
      legend: { 
        display: true,
        position: 'top',
        labels: {
          color: '#24288C',
          font: { size: 12, weight: 'bold' },
          boxWidth: 12,
          boxHeight: 12,
          padding: 15
        },
        maxWidth: 400,
        align: 'start'
      },
      tooltip: {
        enabled: true,
        mode: 'index',
        intersect: false,
        backgroundColor: 'rgba(0, 0, 0, 0.8)',
        titleColor: '#ffffff',
        bodyColor: '#ffffff',
        borderColor: '#ddd',
        borderWidth: 1,
        padding: 10,
        displayColors: true,
        callbacks: {
          title: (context: any) => {
            if (context.length > 0) {
              return context[0].label; // Affiche le mois
            }
            return '';
          },
          label: (context: any) => {
            return `${context.dataset.label}: ${context.parsed.y}`; // Nom type + nombre
          }
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
    datasets: []
  };

  ngOnInit(): void {
    // N'exécuter côté client uniquement
    if (typeof window === 'undefined') {
      return;
    }
    this.loadMetrics();
  }

  private loadMetrics(): void {
    // Récupérer le token du localStorage
    let headers: any = {};
    const token = typeof window !== 'undefined' && window.localStorage ? localStorage.getItem('token') : null;
    
    if (token) {
      headers = { Authorization: `Bearer ${token}` };
    } else {
      return;
    }
    
    // Appel à l'API pour récupérer les métriques (endpoint mis à jour)
    this.http.get<any>('/api/v1/anomalies/metrics', { headers }).subscribe({
      next: (res) => {
        if (!res?.metrics || Object.keys(res.metrics).length === 0) {
          return;
        }
        const metrics = res?.metrics || {};
        
        // Construire les labels pour tous les mois de l'année (Janvier à Décembre)
        const now = new Date();
        const labels: string[] = [];
        const months: string[] = ['Jan', 'Fév', 'Mar', 'Avr', 'Mai', 'Jun', 'Jul', 'Aoû', 'Sep', 'Oct', 'Nov', 'Déc'];
        
        // Afficher janvier à décembre en ordre calendaire
        for (let m = 0; m < 12; m++) {
          labels.push(months[m]);
        }
        this.barChartData.labels = labels;

        // Palette de couleurs FIXES par type (CONFORME à la légende statique)
        const colorByType: { [key: string]: string } = {
          'Montant': '#2463DE',              // Bleu
          'frequence': '#ED6A01',            // Orange
          'Blacklist': '#EF4444',            // Rouge
          'Pays à risque': '#6D5AE0',        // Violet
          'Interdit': '#F59E0B',             // Ambre
          'PEP': '#10B981',                  // Vert
          'Surveillés': '#8B5CF6'            // Pourpre
        };

        // Couleur par défaut si type non trouvé
        const defaultColor = '#9CA3AF';

        // Mapping des types d'alerte pour améliorer l'affichage
        // IMPORTANT: Les clés DOIVENT correspondre EXACTEMENT aux types en base de données
        const typeDisplayMap: { [key: string]: string } = {
          'Pays à risque': 'Pays à Risque',
          'Montant': 'Montant Suspect',
          'frequence': 'Fréquence Anormale',
          'Blacklist': 'Blacklistées',
          'Interdit': 'Interdites',
          'PEP': 'PEP', 
          'Surveillés': 'Comptes Surveillés'
        };

        // Mapper les données des métriques aux datasets
        const buildSeries = (dbType: string, displayLabel: string, color: string) => {
          const arr = new Array(12).fill(0);
          const series = metrics[dbType] || [];
          
          series.forEach((entry: any) => {
            const y = entry.year;
            const m = entry.month - 1;
            
            if (m >= 0 && m < 12) {
              arr[m] = entry.count;
            }
          });
          
          return {
            label: displayLabel,
            data: arr,
            backgroundColor: color,
            borderRadius: 6
          };
        };

        // Créer dynamiquement les datasets pour TOUS les types reçus
        const datasets = Object.keys(metrics).map((dbType) => {
          const displayLabel = typeDisplayMap[dbType] || dbType;
          const color = colorByType[dbType] || defaultColor;
          return buildSeries(dbType, displayLabel, color);
        });
        
        this.barChartData.datasets = datasets;
        
        // Force le graphique à se redessiner avec les nouvelles données
        if (this.chart) {
          this.chart.update();
        }
      },
      error: (err) => {
      }
    });
  }
}

