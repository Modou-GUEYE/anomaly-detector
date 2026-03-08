import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, forkJoin } from 'rxjs';
import { map, tap, switchMap } from 'rxjs/operators';
import jsPDF from 'jspdf';

export interface Report {
  id?: string;
  type: string;
  template_name: string;
  generated_at: Date;
  generated_by: string;
  status: string;
  total_anomalies: number;
  total_transactions: number;
  anomalies_by_type: { [key: string]: number };
  top_risks: {
    type: string;
    severity: string;
    message: string;
    count: number;
  }[];
  metrics: {
    label: string;
    value: number | string;
    unit?: string;
  }[];
  raw_data: {
    anomalies: any[];
    transactions: any[];
    dashboard_metrics: any;
  };
}

export interface ReportTemplate {
  id: string;
  name: string;
  description: string;
  type: 'security' | 'compliance' | 'operational';
  frequency: 'daily' | 'weekly' | 'monthly';
  icon: string;
}

export interface ChartData {
  labels: string[];
  datasets: {
    label: string;
    data: number[];
    backgroundColor?: string[];
    borderColor?: string[];
  }[];
}

export interface RuleViolationData {
  ruleName: string;
  count: number;
  percentage: number;
}

export interface EfficiencyData {
  period: string;
  avgProcessingTime: number;
  resolutionRate: number;
  date: string;
}

@Injectable({
  providedIn: 'root'
})
export class ReportsService {
  private apiUrl = ''; 

  private templates: ReportTemplate[] = [
    {
      id: 'security-daily',
      name: 'Rapport de Sécurité Quotidien',
      description: 'Analyse complète des anomalies et des risques de sécurité détectés',
      type: 'security',
      frequency: 'daily',
      icon: 'shield'
    },
    {
      id: 'compliance-weekly',
      name: 'Rapport de Conformité Hebdomadaire',
      description: 'Évaluation de la conformité réglementaire et des contrôles',
      type: 'compliance',
      frequency: 'weekly',
      icon: 'check-circle'
    },
    {
      id: 'operational-monthly',
      name: 'Rapport Opérationnel Mensuel',
      description: 'Vue d\'ensemble des performances et des métriques opérationnelles',
      type: 'operational',
      frequency: 'monthly',
      icon: 'trending-up'
    }
  ];

  constructor(private http: HttpClient) {}

  private getSystemToken(): Observable<string> {
    const systemRequest = {
      system_id: 'reports-system',
      role: 'admin'
    };
    return this.http.post<{token: string}>(`${this.apiUrl}/api/v1/auth/token`, systemRequest).pipe(
      map(response => response.token)
    );
  }

  getReportTemplates(): ReportTemplate[] {
    return this.templates;
  }

  generateSpecificReport(templateId: string): Observable<Report> {
    // D'abord, obtenons un token système s'il n'y en a pas
    const token = localStorage.getItem('token');
    
    if (token) {
      return this.fetchReportData(templateId);
    } else {
      return this.getSystemToken().pipe(
        tap((systemToken: string) => {
          localStorage.setItem('token', systemToken);
        }),
        switchMap(() => this.fetchReportData(templateId))
      );
    }
  }

  private fetchReportData(templateId: string): Observable<Report> {
    return forkJoin({
      anomalies: this.http.get<any[]>(`${this.apiUrl}/api/v1/anomalies`),
      transactions: this.http.get<any[]>(`${this.apiUrl}/api/v1/transactions`),
      dashboard: this.http.get<any>(`${this.apiUrl}/api/v1/dashboard/summary`),
      paysRisque: this.http.get<any[]>(`${this.apiUrl}/api/v1/pays-risque`)
    }).pipe(
      map(data => {
        const template = this.templates.find(t => t.id === templateId);
        if (!template) {
          throw new Error(`Template ${templateId} non trouvé`);
        }

        return this.processReportData(data, template);
      })
    );
  }

  private processReportData(data: any, template: ReportTemplate): Report {
    const anomalies = data.anomalies || [];
    const transactions = data.transactions || [];
    const dashboardData = data.dashboard || {};
    const paysRisque = data.paysRisque || [];

    // Analyse des anomalies par type
    const anomaliesByType: { [key: string]: number } = {};
    anomalies.forEach((anomaly: any) => {
      const type = anomaly.type || 'Unknown';
      anomaliesByType[type] = (anomaliesByType[type] || 0) + 1;
    });

    // Analyse des risques
    const topRisks = this.analyzeRisks(anomalies, paysRisque);

    // Métriques
    const metrics = [
      { label: 'Total Anomalies', value: anomalies.length },
      { label: 'Total Transactions', value: transactions.length },
      { label: 'Pays à Risque', value: paysRisque.length },
      { label: 'Taux de Détection', value: anomalies.length > 0 ? ((anomalies.length / Math.max(transactions.length, 1)) * 100).toFixed(2) : '0', unit: '%' }
    ];

    return {
      id: this.generateReportId(),
      type: template.type,
      template_name: template.name,
      generated_at: new Date(),
      generated_by: 'Système',
      status: 'completed',
      total_anomalies: anomalies.length,
      total_transactions: transactions.length,
      anomalies_by_type: anomaliesByType,
      top_risks: topRisks,
      metrics: metrics,
      raw_data: {
        anomalies: anomalies,
        transactions: transactions,
        dashboard_metrics: dashboardData
      }
    };
  }

  private analyzeRisks(anomalies: any[], paysRisque: any[]): any[] {
    const risks = [];

    // Risques basés sur les anomalies
    const riskTypes = ['CRITICAL', 'HIGH', 'MEDIUM'];
    const anomaliesByRisk = anomalies.reduce((acc: any, anomaly: any) => {
      const severity = anomaly.severity || 'MEDIUM';
      if (!acc[severity]) acc[severity] = [];
      acc[severity].push(anomaly);
      return acc;
    }, {});

    riskTypes.forEach(severity => {
      if (anomaliesByRisk[severity] && anomaliesByRisk[severity].length > 0) {
        const count = anomaliesByRisk[severity].length;
        const examples = anomaliesByRisk[severity].slice(0, 3);
        
        risks.push({
          type: 'Anomalies de Sécurité',
          severity: severity,
          message: `${count} anomalie(s) de niveau ${severity.toLowerCase()} détectée(s). Exemples: ${examples.map((a: any) => a.description || a.type).join(', ')}`,
          count: count
        });
      }
    });

    // Risques basés sur les pays
    if (paysRisque.length > 0) {
      risks.push({
        type: 'Risques Géographiques',
        severity: 'HIGH',
        message: `${paysRisque.length} pays à risque identifiés dans la base de données`,
        count: paysRisque.length
      });
    }

    return risks.slice(0, 5); // Top 5 risques
  }

  private generateReportId(): string {
    return `RPT-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  exportToPDF(report: Report): void {
    const doc = new jsPDF();
    
    // En-tête
    this.addPDFHeader(doc, report);
    
    // Résumé exécutif
    this.addExecutiveSummary(doc, report);
    
    // Métriques clés
    this.addMetrics(doc, report);
    
    // Anomalies par type
    this.addAnomaliesByType(doc, report);
    
    // Top risques
    this.addTopRisks(doc, report);
    
    // Pied de page
    this.addPDFFooter(doc);
    
    doc.save(`${report.template_name}_${new Date().toISOString().split('T')[0]}.pdf`);
  }

  exportToExcel(report: Report): void {
    const lines: string[] = [];
    lines.push(`Rapport;${this.escapeCsvField(report.template_name)}`);
    lines.push(`Date de génération;${this.escapeCsvField(report.generated_at.toLocaleDateString('fr-FR'))}`);
    lines.push(`Statut;${this.escapeCsvField(report.status)}`);
    lines.push(`Total anomalies;${report.total_anomalies}`);
    lines.push(`Total transactions;${report.total_transactions}`);
    lines.push('');

    lines.push('Anomalies par type;Nombre');
    Object.entries(report.anomalies_by_type).forEach(([type, count]) => {
      lines.push(`${this.escapeCsvField(type)};${count}`);
    });
    lines.push('');

    lines.push('Type risque;Sévérité;Message;Nombre');
    report.top_risks.forEach(risk => {
      lines.push([
        this.escapeCsvField(risk.type),
        this.escapeCsvField(risk.severity),
        this.escapeCsvField(risk.message),
        String(risk.count)
      ].join(';'));
    });
    lines.push('');

    if (report.raw_data.anomalies.length > 0) {
      lines.push('Données brutes anomalies');
      const keys = Object.keys(report.raw_data.anomalies[0]);
      lines.push(keys.map(k => this.escapeCsvField(k)).join(';'));
      report.raw_data.anomalies.forEach((row: any) => {
        lines.push(keys.map(k => this.escapeCsvField(String(row?.[k] ?? ''))).join(';'));
      });
    }

    const csvContent = '\ufeff' + lines.join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const fileName = `${report.template_name}_${new Date().toISOString().split('T')[0]}.csv`;
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }

  private addPDFHeader(doc: jsPDF, report: Report): void {
    doc.setFontSize(20);
    doc.setTextColor(46, 108, 242);
    doc.text(report.template_name, 20, 30);
    
    doc.setFontSize(12);
    doc.setTextColor(60, 60, 60);
    doc.text(`Généré le: ${report.generated_at.toLocaleDateString('fr-FR')}`, 20, 45);
    doc.text(`Statut: ${report.status}`, 120, 45);
    
    // Ligne de séparation
    doc.setDrawColor(200, 200, 200);
    doc.line(20, 55, 190, 55);
  }

  private addExecutiveSummary(doc: jsPDF, report: Report): void {
    let yPosition = 70;
    
    doc.setFontSize(16);
    doc.setTextColor(46, 108, 242);
    doc.text('RÉSUMÉ EXÉCUTIF', 20, yPosition);
    yPosition += 15;
    
    doc.setFontSize(11);
    doc.setTextColor(60, 60, 60);
    
    const summaryText = [
      `Ce rapport présente une analyse complète des anomalies détectées.`,
      `${report.total_anomalies} anomalies ont été identifiées sur ${report.total_transactions} transactions analysées.`,
      `${report.top_risks.length} risques majeurs nécessitent une attention particulière.`
    ];
    
    summaryText.forEach(text => {
      const splitText = doc.splitTextToSize(text, 150);
      doc.text(splitText, 20, yPosition);
      yPosition += splitText.length * 6 + 5;
    });
  }

  private addMetrics(doc: jsPDF, report: Report): void {
    let yPosition = 140;
    
    doc.setFontSize(14);
    doc.setTextColor(46, 108, 242);
    doc.text('MÉTRIQUES CLÉS', 20, yPosition);
    yPosition += 15;
    
    doc.setFontSize(10);
    doc.setTextColor(60, 60, 60);
    
    report.metrics.forEach(metric => {
      const value = metric.unit ? `${metric.value} ${metric.unit}` : metric.value.toString();
      doc.text(`• ${metric.label}: ${value}`, 20, yPosition);
      yPosition += 8;
    });
  }

  private addAnomaliesByType(doc: jsPDF, report: Report): void {
    doc.addPage();
    let yPosition = 30;
    
    doc.setFontSize(14);
    doc.setTextColor(46, 108, 242);
    doc.text('ANOMALIES PAR TYPE', 20, yPosition);
    yPosition += 15;
    
    doc.setFontSize(10);
    doc.setTextColor(60, 60, 60);
    
    Object.entries(report.anomalies_by_type).forEach(([type, count]) => {
      doc.text(`• ${type}: ${count} occurrence(s)`, 20, yPosition);
      yPosition += 8;
    });
  }

  private addTopRisks(doc: jsPDF, report: Report): void {
    doc.addPage();
    let yPosition = 30;
    
    doc.setFontSize(14);
    doc.setTextColor(46, 108, 242);
    doc.text('PRINCIPAUX RISQUES IDENTIFIÉS', 20, yPosition);
    yPosition += 15;
    
    doc.setFontSize(10);
    doc.setTextColor(60, 60, 60);
    
    report.top_risks.forEach((risk, index) => {
      // Couleur selon la sévérité
      const severityColor = risk.severity === 'CRITICAL' ? [255, 0, 0] :
                           risk.severity === 'HIGH' ? [255, 140, 0] :
                           [255, 193, 7];
      
      doc.setTextColor(46, 108, 242);
      doc.text(`${index + 1}.`, 20, yPosition);
      
      doc.setFontSize(11);
      doc.setTextColor(60, 60, 60);
      doc.text(`Type: ${risk.type}`, 30, yPosition);
      yPosition += 6;
      
      doc.setTextColor(severityColor[0], severityColor[1], severityColor[2]);
      doc.text(`Sévérité: ${risk.severity.toUpperCase()}`, 30, yPosition);
      yPosition += 6;
      
      doc.setFontSize(10);
      doc.setTextColor(60, 60, 60);
      const splitMessage = doc.splitTextToSize(`Message: ${risk.message}`, 150);
      doc.text(splitMessage, 30, yPosition);
      yPosition += splitMessage.length * 5 + 10;
    });
  }

  private addPDFFooter(doc: jsPDF): void {
    const pageCount = doc.getNumberOfPages();
    
    for (let i = 1; i <= pageCount; i++) {
      doc.setPage(i);
      
      // Ligne de séparation
      doc.setDrawColor(200, 200, 200);
      doc.line(20, 280, 190, 280);
      
      // Texte du pied de page
      doc.setFontSize(8);
      doc.setTextColor(120, 120, 120);
      doc.text('Système de Détection d\'Anomalies - Confidentiel', 20, 285);
      doc.text(`Page ${i} sur ${pageCount}`, 190, 285, { align: 'right' });
      doc.text(`Généré le ${new Date().toLocaleDateString('fr-FR')}`, 20, 290);
    }
  }

  private escapeCsvField(value: string): string {
    if (value.includes(';') || value.includes('\n') || value.includes('"')) {
      return `"${value.replace(/"/g, '""')}"`;
    }
    return value;
  }

  /**
   * Récupère les données des règles les plus violées depuis le dashboard
   */
  getRuleViolationsData(period: 'month' | 'quarter' | 'year' = 'year'): Observable<RuleViolationData[]> {
    const params = { period };
    const url = `${this.apiUrl}/api/v1/reports/rule-violations`;
    return this.http.get<RuleViolationData[]>(url, { params });
  }

  /**
   * Récupère les données d'efficacité du traitement
   */
  getEfficiencyData(period: 'month' | 'quarter' | 'year' = 'year'): Observable<EfficiencyData[]> {
    const params = { period };
    const url = `${this.apiUrl}/api/v1/reports/efficiency`;
    return this.http.get<any[]>(url, { params }).pipe(
      map(data => data.map(item => ({
        period: item.period,
        avgProcessingTime: item.avgProcessingTime || 0,
        resolutionRate: item.resolutionRate || 0,
        date: item.date
      })))
    );
  }

  /**
   * Récupère les données de performance pour les KPIs
   */
  getPerformanceKPIs(dateStart?: string, dateEnd?: string): Observable<any> {
    let params: any = {};
    if (dateStart) params.date_start = dateStart;
    if (dateEnd) params.date_end = dateEnd;
    
    return this.http.get<any>(`${this.apiUrl}/api/v1/reports/performance`, { params });
  }


  /**
   * Récupère les métriques de conformité LBC/FT
   */
  getComplianceMetrics(): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}/api/v1/reports/compliance`);
  }

  /**
   * Récupère le résumé complet du dashboard avec toutes les métriques
   */
  getDashboardSummary(dateDebut?: string, dateFin?: string): Observable<any> {
    let params: any = {};
    if (dateDebut) params.dateDebut = dateDebut;
    if (dateFin) params.dateFin = dateFin;
    return this.http.get<any>(`${this.apiUrl}/api/v1/dashboard/summary`, { params });
  }

  /**
   * Récupère les informations d'un utilisateur par son ID
   */
  getUserById(userId: string): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}/api/v1/users/${userId}`);
  }
}