import { Component, OnInit, OnDestroy, ViewChild, ElementRef, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, finalize } from 'rxjs/operators';
import { Chart, ChartConfiguration, ChartType, registerables } from 'chart.js';
import { ReportsService, RuleViolationData, EfficiencyData } from '../../services/reports.service';


@Component({
  selector: 'app-reports',
  standalone: true,
  templateUrl: './reports.html',
  styleUrls: ['./reports.scss'],
  imports: [CommonModule, FormsModule]
})
export class ReportsComponent implements OnInit, OnDestroy, AfterViewInit {
  @ViewChild('chartReglesViolees') chartReglesViolees!: ElementRef<HTMLCanvasElement>;
  @ViewChild('chartEfficacite') chartEfficacite!: ElementRef<HTMLCanvasElement>;

  loading = false;
  error = '';
  exporting = false; // état export en cours (PDF/Excel)

  // Charts
  private chartRegles: Chart | null = null;
  private chartEffic: Chart | null = null;

  // Performance data
  performanceData = {
    anomaliesGenerees: 0,
    anomaliesTraitees: 0,
    transactionsAnalysees: 0
  };

  // Regulatory data
  regulatoryData = {
    transactionsBloquees: 0,
    comptesSurveilles: 0
  };

  // Filters
  filters = {
    dateDebut: '',
    dateFin: ''
  };

  // Current filters for charts
  currentFilters: { period: 'month' | 'quarter' | 'year'; typeRule: 'all' | 'montant' | 'frequence' | 'pays' | 'blacklist' | 'pep' | 'surveilles' | 'interdit' } = {
    period: 'month',
    typeRule: 'all'
  };

  // Loading states for different sections
  loadingStates = {
    performance: false,
    charts: false
  };

  // Real data
  realRuleViolationsData: RuleViolationData[] = [];
  realEfficiencyData: EfficiencyData[] = [];

  private destroy$ = new Subject<void>();

  constructor(private reportsService: ReportsService) {
    Chart.register(...registerables);
  }

  ngOnInit() {
    this.initializeFilters();
    this.loadRealData();
  }

  ngAfterViewInit() {
    // Création directe des charts une fois les ViewChild dispos
    this.createChartsIfReady();
  }

  /**
   * Charge les vraies données depuis l'API
   */
  loadRealData() {
  this.loadComplianceData();
  this.loadChartData();
  }

  /**
   * Charge les données de conformité avec filtres de dates
   */
  loadComplianceData() {
    this.reportsService.getDashboardSummary(this.filters.dateDebut, this.filters.dateFin)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (data: any) => {
          if (data) {
            this.performanceData = {
              anomaliesGenerees: data.anomalies_count || 0,
              anomaliesTraitees: data.anomalies_treated || 0,
              transactionsAnalysees: data.transactions_count || 0
            };

            this.regulatoryData = {
              transactionsBloquees: data.transactions_blocked || 0,
              comptesSurveilles: data.monitored_accounts || 0
            };
          }
        },
        error: () => {
          this.setError('Erreur chargement métriques tableau de bord');
        }
      });
  }

  /**
   * Charge les données des graphiques
   */
  loadChartData() {
    this.loadingStates.charts = true;
    
    // Règles violées
    this.reportsService.getRuleViolationsData(this.currentFilters.period)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (data: any[]) => {
          // Le backend gère déjà le fallback année -> mois automatiquement
          // On affiche les données directement
          this.processRuleViolationsData(data || []);
        },
        error: () => {
          this.setError(`Erreur chargement règles violées (période: ${this.currentFilters.period})`);
        }
      });

  // Efficacité du traitement
    this.reportsService.getEfficiencyData(this.currentFilters.period)
      .pipe(
        takeUntil(this.destroy$),
        finalize(() => this.loadingStates.charts = false)
      )
      .subscribe({
        next: (data: EfficiencyData[]) => {
          this.realEfficiencyData = data;
          if (this.chartEffic) {
            this.updateEfficiencyChartWithRealData();
          } else {
            this.createChartsIfReady();
          }
        },
        error: () => {
          this.setError('Erreur chargement données efficacité');
        }
      });
  }

  /**
   * Traite les données des règles violées (centralisé)
   */
  private processRuleViolationsData(data: any[]) {
    const transformedData: RuleViolationData[] = (data || []).map((item: any) => ({
      ruleName: item.rule_name || item.ruleName || 'Règle inconnue',
      count: item.count || 0,
      percentage: item.percentage || 0
    }));
    
    this.realRuleViolationsData = transformedData;

    
    if (this.chartRegles) {
      this.updateRuleChartWithRealData();
    } else {
      this.createChartsIfReady();
    }
  }

  ngOnDestroy() {
    if (this.chartRegles) {
      this.chartRegles.destroy();
    }
    if (this.chartEffic) {
      this.chartEffic.destroy();
    }
    this.destroy$.next();
    this.destroy$.complete();
  }

  /**
   * Initialise les filtres avec les dates par défaut
   */
  initializeFilters() {
    const today = new Date();
    const monthAgo = new Date(today.getFullYear(), today.getMonth() - 1, today.getDate());
    
    this.filters.dateFin = today.toISOString().split('T')[0];
    this.filters.dateDebut = monthAgo.toISOString().split('T')[0];
  }

  /**
   * Crée les graphiques si les éléments DOM sont prêts
   */
  createChartsIfReady() {
    if (this.chartReglesViolees && this.chartEfficacite) {
      this.createRulesViolationChart();
      this.createEfficiencyChart();
      
      if (this.realRuleViolationsData !== null) {
        this.updateRuleChartWithRealData();
      }
      if (this.realEfficiencyData.length > 0) {
        this.updateEfficiencyChartWithRealData();
      }
    }
  }

  /**
   * Crée le graphique des règles violées (bar chart)
   */
  private createRulesViolationChart() {
    const ctx = this.chartReglesViolees.nativeElement.getContext('2d');
    if (!ctx) return;

    if (this.chartRegles) {
      this.chartRegles.destroy();
    }

    const config: ChartConfiguration = {
      type: 'bar' as ChartType,
      data: {
        labels: ['Chargement...'],
        datasets: [{
          label: 'Nombre d\'alertes',
          data: [0],
          backgroundColor: ['#E5E7EB'],
          borderColor: ['#D1D5DB'],
          borderWidth: 2,
          borderRadius: 6,
          borderSkipped: false,
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: { enabled: false }
        },
        scales: {
          y: {
            beginAtZero: true,
            grid: { color: 'rgba(0, 0, 0, 0.1)' },
            ticks: { color: '#9CA3AF' }
          },
          x: {
            grid: { display: false },
            ticks: { color: '#9CA3AF' }
          }
        }
      }
    };
    
    this.chartRegles = new Chart(ctx, config);
  }

  /**
   * Crée le graphique d'efficacité du traitement (line chart)
   */
  private createEfficiencyChart() {
    const ctx = this.chartEfficacite.nativeElement.getContext('2d');
    if (!ctx) return;

    if (this.chartEffic) {
      this.chartEffic.destroy();
    }

  // ...existing code...
    const hasData = this.realEfficiencyData.length > 0;
    const labels = hasData ? this.realEfficiencyData.map(item => item.period) : ['Aucune donnée'];
    const delaiData = hasData ? this.realEfficiencyData.map(item => item.avgProcessingTime) : [0];
    const tauxData = hasData ? this.realEfficiencyData.map(item => item.resolutionRate) : [0];

    const config: ChartConfiguration = {
      type: 'line' as ChartType,
      data: {
        labels: labels,
        datasets: [
          {
            label: 'Délai Moyen (jours)',
            data: delaiData,
            borderColor: '#2E6CF2',
            backgroundColor: 'rgba(46, 108, 242, 0.1)',
            borderWidth: 3,
            fill: true,
            tension: 0.4,
            pointBackgroundColor: '#2E6CF2',
            pointBorderColor: '#fff',
            pointBorderWidth: 2,
            pointRadius: 5,
            pointHoverRadius: 7
          },
          {
            label: 'Taux de Résolution (%)',
            data: tauxData,
            borderColor: '#10B981',
            backgroundColor: 'rgba(16, 185, 129, 0.1)',
            borderWidth: 3,
            fill: true,
            tension: 0.4,
            pointBackgroundColor: '#10B981',
            pointBorderColor: '#fff',
            pointBorderWidth: 2,
            pointRadius: 5,
            pointHoverRadius: 7,
            yAxisID: 'y1'
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: {
          mode: 'index',
          intersect: false,
        },
        hover: {
          mode: 'index',
          intersect: false,
        },
        plugins: {
          legend: {
            position: 'top',
            labels: {
              usePointStyle: true,
              padding: 20,
              color: '#374151'
            }
          },
          tooltip: {
            enabled: true,
            backgroundColor: 'rgba(0, 0, 0, 0.8)',
            titleColor: '#fff',
            bodyColor: '#fff',
            borderColor: 'rgba(255, 255, 255, 0.1)',
            borderWidth: 1,
            cornerRadius: 8,
            padding: 12,
            displayColors: true,
            mode: 'index',
            intersect: false,
            callbacks: {
              title: (context) => {
                return `Période: ${context[0].label}`;
              },
              label: (context) => {
                const dataset = context.dataset.label;
                const value = context.parsed.y;
                if (dataset?.includes('Délai')) {
                  return `${dataset}: ${value} jours`;
                } else if (dataset?.includes('Taux')) {
                  return `${dataset}: ${value}%`;
                }
                return `${dataset}: ${value}`;
              },
              footer: (context) => {
                if (hasData && context.length > 0) {
                  const index = context[0].dataIndex;
                  const data = this.realEfficiencyData[index];
                  if (data && data.date) {
                    const formattedDate = new Date(data.date).toLocaleDateString('fr-FR', {
                      day: '2-digit',
                      month: '2-digit',
                      year: 'numeric'
                    });
                    return `Date: ${formattedDate}`;
                  }
                }
                return '';
              }
            }
          }
        },
        scales: {
          x: {
            grid: { display: false },
            ticks: { color: '#6B7280' }
          },
          y: {
            type: 'linear',
            display: true,
            position: 'left',
            beginAtZero: true,
            grid: { color: 'rgba(0, 0, 0, 0.1)' },
            ticks: {
              color: '#6B7280',
              callback: function(value) {
                return value + ' jours';
              }
            }
          },
          y1: {
            type: 'linear',
            display: true,
            position: 'right',
            beginAtZero: true,
            max: 100,
            grid: { drawOnChartArea: false },
            ticks: {
              color: '#6B7280',
              callback: function(value) {
                return value + '%';
              }
            }
          }
        },
        animation: {
          duration: 1500,
          easing: 'easeInOutQuart'
        }
      }
    };

    this.chartEffic = new Chart(ctx, config);
  }

  /**
   * Met à jour le graphique des règles avec les vraies données
   */
  private updateRuleChartWithRealData() {
    if (!this.chartRegles) return;

    if (this.realRuleViolationsData.length === 0) {
      this.chartRegles.data.labels = ['Aucune donnée'];
      this.chartRegles.data.datasets[0].data = [0];
      this.chartRegles.data.datasets[0].backgroundColor = ['#e0e0e0'];
      this.chartRegles.update();
      return;
    }
    // Filtrage souple: si type spécifique, garder ceux dont le nom contient le type (minuscule)
    let filteredData = this.realRuleViolationsData;
    if (this.currentFilters.typeRule !== 'all') {
      const needle = this.currentFilters.typeRule.toLowerCase();
      filteredData = this.realRuleViolationsData.filter(item => item.ruleName.toLowerCase().includes(needle));
    }

    const labels = filteredData.map(item => item.ruleName);
    const data = filteredData.map(item => item.count);

    // Mettre à jour les données
    this.chartRegles.data.labels = labels;
    this.chartRegles.data.datasets[0].data = data;
    
    // Mettre à jour les couleurs
    if (filteredData.length > 0) {
      const colors = this.getColorsForRules(filteredData);
      this.chartRegles.data.datasets[0].backgroundColor = colors.background;
      this.chartRegles.data.datasets[0].borderColor = colors.border;
      this.chartRegles.options.plugins!.tooltip!.enabled = true;
    }
    
    this.chartRegles.update('active');
  }

  /**
   * Met à jour le graphique d'efficacité avec les vraies données
   */
  private updateEfficiencyChartWithRealData() {
    if (!this.chartEffic || this.realEfficiencyData.length === 0) return;

    const labels = this.realEfficiencyData.map(item => item.period);
    const delaiData = this.realEfficiencyData.map(item => item.avgProcessingTime);
    const tauxData = this.realEfficiencyData.map(item => item.resolutionRate);

    this.chartEffic.data.labels = labels;
    this.chartEffic.data.datasets[0].data = delaiData;
    this.chartEffic.data.datasets[1].data = tauxData;
    this.chartEffic.update('active');
  }

  /**
   * Met à jour les graphiques en fonction des filtres
   */
  updateCharts() {
    if (this.chartRegles) {
      this.updateRulesChart();
    }
    if (this.chartEffic) {
      this.updateEfficiencyChart();
    }
  }

  /**
   * Met à jour le graphique des règles avec les vraies données filtrées
   */
  private updateRulesChart() {
    // Redirection vers la méthode unifiée
    this.updateRuleChartWithRealData();
  }

  /**
   * Détermine les couleurs pour les règles violées en fonction de leur type
   */
  private getColorsForRules(data: RuleViolationData[]): { background: string[], border: string[] } {
    const defaultColors = [
      { bg: '#2463DE', border: '#1D4ED8' }, // Bleu principal
      { bg: '#ED6A01', border: '#C2410C' }, // Orange
      { bg: '#EF4444', border: '#DC2626' }, // Rouge
      { bg: '#6D5AE0', border: '#5B21B6' }, // Violet
      { bg: '#F59E0B', border: '#D97706' }, // Ambre
      { bg: '#10B981', border: '#059669' }, // Vert
      { bg: '#8B5CF6', border: '#7C3AED' }, // Indigo
      { bg: '#EC4899', border: '#DB2777' }, // Rose
      { bg: '#06B6D4', border: '#0891B2' }, // Cyan
      { bg: '#9CA3AF', border: '#6B7280' }  // Gris
    ];

    const backgroundColors: string[] = [];
    const borderColors: string[] = [];

    data.forEach((item, index) => {
      const colors = defaultColors[index % defaultColors.length];
      backgroundColors.push(colors.bg);
      borderColors.push(colors.border);
    });

    return { background: backgroundColors, border: borderColors };
  }

  /**
   * Met à jour le graphique d'efficacité avec les vraies données filtrées
   */
  private updateEfficiencyChart() {
    if (!this.chartEffic || this.realEfficiencyData.length === 0) return;

    // Utiliser les vraies données filtrées selon la période
    const filteredData = this.realEfficiencyData;
    
    this.chartEffic.data.labels = filteredData.map(item => item.period);
    this.chartEffic.data.datasets[0].data = filteredData.map(item => item.avgProcessingTime);
    this.chartEffic.data.datasets[1].data = filteredData.map(item => item.resolutionRate);
    this.chartEffic.update('active');
  }

  /**
   * Gère les changements de filtres
   */
  onFiltersChange() {
  this.loadComplianceData(); // Recharge les métriques avec les nouvelles dates
  }

  /**
   * Retourne le label de la période sélectionnée
   */
  getPeriodLabel(): string {
    if (this.filters.dateDebut && this.filters.dateFin) {
      const debut = new Date(this.filters.dateDebut);
      const fin = new Date(this.filters.dateFin);
      return `Du ${debut.toLocaleDateString()} au ${fin.toLocaleDateString()}`;
    }
    return 'Période actuelle';
  }

  /**
   * Change la période pour les graphiques de performance
   */
  changePeriod(period: 'month' | 'quarter' | 'year') {
    this.currentFilters.period = period;
    this.loadChartData(); // Recharge les vraies données pour la nouvelle période
  }

  /**
   * Change le type de règle pour le graphique des règles violées
   */
  changeRuleType(type: 'all' | 'montant' | 'frequence' | 'pays' | 'blacklist' | 'pep' | 'surveilles' | 'interdit') {
    this.currentFilters.typeRule = type;
    this.updateRuleChartWithRealData();
  }

  /**
   * Force le rechargement des données en temps réel
   */
  refreshData() {
    this.loadRealData();
  }

  /**
   * Retourne le label du type d'alerte
   */
  getTypeAlerteLabel(type: string): string {
    return type;
  }

  /**
   * Retourne le label du statut
   */
  getStatutLabel(statut: string): string {
    return statut;
  }



  /**
   * Exporte les données en PDF
   */
  exportToPDF() {
    if (this.exporting) return;
    this.exporting = true;
    import('jspdf').then(jsPDFModule => {
      const jsPDF = jsPDFModule.default;
      const doc = new jsPDF();
      let y = 20;
      // TITRE PRINCIPAL
      doc.setFont('helvetica', 'bold');
      doc.setTextColor(46, 108, 242); // Bleu branding
      doc.setFontSize(18);
      doc.text('RAPPORT LCB/FT - SYNTHÈSE OPÉRATIONNELLE', 15, y);
      y += 10;
      doc.setDrawColor(200, 200, 200);
      doc.line(15, y, 195, y);
      y += 6;
      // FILTRES
      doc.setFont('helvetica', 'bold');
      doc.setFontSize(12);
      doc.setTextColor(55, 65, 81); // Gris foncé
      doc.text('FILTRES UTILISÉS', 15, y);
      y += 7;
      doc.setFont('helvetica', 'normal');
      doc.setFontSize(10);
      // Période analysée
      let periodeLabel = '';
      switch (this.currentFilters.period) {
        case 'month': periodeLabel = 'Mois'; break;
        case 'quarter': periodeLabel = 'Trimestre'; break;
        case 'year': periodeLabel = 'Année'; break;
        default: periodeLabel = this.currentFilters.period;
      }
      // Préparer le label de la règle sélectionnée
      let typeRuleLabel = '';
      switch (this.currentFilters.typeRule) {
        case 'all': typeRuleLabel = 'Toutes'; break;
        case 'montant': typeRuleLabel = 'Montant'; break;
        case 'frequence': typeRuleLabel = 'Fréquence'; break;
        case 'pays': typeRuleLabel = 'Pays à risque'; break;
        case 'blacklist': typeRuleLabel = 'Blacklist'; break;
        case 'pep': typeRuleLabel = 'PEP'; break;
        case 'surveilles': typeRuleLabel = 'Surveillés'; break;
        case 'interdit': typeRuleLabel = 'Interdit'; break;
        default: typeRuleLabel = this.currentFilters.typeRule;
      }
      doc.text(`Période Analysée : ${periodeLabel}`, 15, y);
      y += 5;
      if (this.filters.dateDebut) {
        doc.text(`Date de Début : ${this.filters.dateDebut}`, 15, y);
        y += 5;
      }
      if (this.filters.dateFin) {
        doc.text(`Date de Fin : ${this.filters.dateFin}`, 15, y);
        y += 5;
      }
      doc.text(`Portée des Règles : ${typeRuleLabel}`, 15, y);
      y += 8;
      doc.setDrawColor(220, 220, 220);
      doc.line(15, y, 195, y);
      y += 6;
      // INDICATEURS CLÉS
      doc.setFont('helvetica', 'bold');
      doc.setFontSize(12);
      doc.setTextColor(46, 108, 242);
      doc.text('INDICATEURS CLÉS (KPI)', 15, y);
      y += 7;
      doc.setFont('helvetica', 'normal');
      doc.setFontSize(10);
      doc.setTextColor(55, 65, 81);
      doc.text(`Anomalies Générées : ${this.performanceData.anomaliesGenerees}`, 15, y);
      y += 5;
      doc.text(`Anomalies Traitées : ${this.performanceData.anomaliesTraitees}`, 15, y);
      y += 5;
      doc.text(`Transactions Analysées : ${this.performanceData.transactionsAnalysees}`, 15, y);
      y += 5;
      doc.text(`Transactions Bloquées : ${this.regulatoryData.transactionsBloquees}`, 15, y);
      y += 5;
      doc.text(`Comptes Sous Surveillance : ${this.regulatoryData.comptesSurveilles}`, 15, y);
      y += 8;
      doc.setDrawColor(220, 220, 220);
      doc.line(15, y, 195, y);
      y += 6;
      // RÉPARTITION DES VIOLATIONS DE RÈGLES
      doc.setFont('helvetica', 'bold');
      doc.setFontSize(12);
      doc.setTextColor(46, 108, 242);
      doc.text('RÉPARTITION DES VIOLATIONS DE RÈGLES', 15, y);
      y += 7;
      doc.setFont('helvetica', 'bold');
      doc.setFontSize(10);
      doc.setFillColor(243, 244, 246); // Gris très clair
      doc.rect(15, y - 4, 180, 7, 'F');
      doc.setTextColor(55, 65, 81);
      doc.text('Règle', 18, y);
      doc.text('Nombre', 80, y);
      y += 6;
      doc.setFont('helvetica', 'normal');
      doc.setTextColor(55, 65, 81);
      const filteredData = this.currentFilters.typeRule === 'all'
        ? this.realRuleViolationsData
        : this.realRuleViolationsData.filter(item => item.ruleName === this.currentFilters.typeRule);
      if (filteredData.length === 0) {
        doc.text('Aucune donnée.', 18, y);
        y += 6;
      } else {
        filteredData.forEach(item => {
          doc.text(String(item.ruleName), 18, y);
          doc.text(String(item.count), 80, y);
          y += 6;
        });
      }
      y += 6;
      doc.setDrawColor(220, 220, 220);
      doc.line(15, y, 195, y);
      y += 6;
      // EFFICACITÉ ET DÉLAIS DE TRAITEMENT
      doc.setFont('helvetica', 'bold');
      doc.setFontSize(12);
      doc.setTextColor(46, 108, 242);
      doc.text('EFFICACITÉ ET DÉLAIS DE TRAITEMENT', 15, y);
      y += 7;
      doc.setFont('helvetica', 'bold');
      doc.setFontSize(10);
      doc.setFillColor(243, 244, 246);
      doc.rect(15, y - 4, 180, 7, 'F');
      doc.setTextColor(55, 65, 81);
      doc.text('Période', 18, y);
      doc.text('Délai Moyen (jours)', 60, y);
      doc.text('Taux de Résolution (%)', 120, y);
      y += 6;
      doc.setFont('helvetica', 'normal');
      if (this.realEfficiencyData.length === 0) {
        doc.text('Aucune donnée.', 18, y);
        y += 6;
      } else {
        this.realEfficiencyData.forEach(item => {
          doc.text(String(item.period), 18, y);
          doc.text(String(item.avgProcessingTime), 60, y);
          doc.text(String(item.resolutionRate), 120, y);
          y += 6;
        });
      }
      y += 8;
      doc.setDrawColor(200, 200, 200);
      doc.line(15, y, 195, y);
      y += 8;
      doc.setFont('helvetica', 'bold');
      doc.setFontSize(12);
      doc.setTextColor(55, 65, 81);
      doc.text('Fin du Rapport.', 15, y);
      doc.save(`rapport_lbcft_${new Date().toISOString().split('T')[0]}.pdf`);
      this.exporting = false;
    });
  }

  exportToExcel() {
    if (this.exporting) return;
    this.exporting = true;
    // Préparer les données pour l'export Excel
    const data = [
      // En-têtes
      ['RAPPORT LCB/FT - SYNTHÈSE OPÉRATIONNELLE'],
      [''],
      ['FILTRES UTILISÉS'],
      [`Période Analysée: ${this.currentFilters.period}`],
      [`Date de Début: ${this.filters.dateDebut || 'N/A'}`],
      [`Date de Fin: ${this.filters.dateFin || 'N/A'}`],
      [''],
      ['INDICATEURS CLÉS (KPI)'],
      [`Anomalies Générées: ${this.performanceData.anomaliesGenerees}`],
      [`Anomalies Traitées: ${this.performanceData.anomaliesTraitees}`],
      [`Transactions Analysées: ${this.performanceData.transactionsAnalysees}`],
      [`Transactions Bloquées: ${this.regulatoryData.transactionsBloquees}`],
      [`Comptes Sous Surveillance: ${this.regulatoryData.comptesSurveilles}`],
      [''],
      ['RÉPARTITION DES VIOLATIONS DE RÈGLES'],
      ['Règle', 'Nombre']
    ];

    // Ajouter les données de violations
    const filteredData = this.currentFilters.typeRule === 'all'
      ? this.realRuleViolationsData
      : this.realRuleViolationsData.filter(item => item.ruleName === this.currentFilters.typeRule);
    
    if (filteredData.length === 0) {
      data.push(['Aucune donnée']);
    } else {
      filteredData.forEach(item => {
        data.push([item.ruleName, item.count.toString()]);
      });
    }

    data.push(['']);
    data.push(['EFFICACITÉ ET DÉLAIS DE TRAITEMENT']);
    data.push(['Période', 'Délai Moyen (jours)', 'Taux de Résolution (%)']);

    // Ajouter les données d'efficacité
    if (this.realEfficiencyData.length === 0) {
      data.push(['Aucune donnée']);
    } else {
      this.realEfficiencyData.forEach(item => {
        data.push([item.period, item.avgProcessingTime.toString(), item.resolutionRate.toString()]);
      });
    }

    // Créer le CSV
    const csvContent = data.map(row => row.join(';')).join('\n');
    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `rapport_lbcft_${new Date().toISOString().split('T')[0]}.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    this.exporting = false;
  }

  private setError(message: string) {
    this.error = message;
  }
}