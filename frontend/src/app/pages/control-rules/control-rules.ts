import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ControlRulesService, ControlRule, ProfilClient } from '../../services/control-rules.service';
import { AuthService } from '../../services/auth.service';
import { ConfigService } from '../../services/config.service';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-control-rules',
  standalone: true,
  templateUrl: './control-rules.html',
  styleUrls: ['./control-rules.scss'],
  imports: [CommonModule, FormsModule, MatIconModule]
})
export class ControlRulesComponent implements OnInit {
  loading = true;
  error = '';
  isAdmin = false;
  adminOnlyMessage = 'Cette action est réservée aux administrateurs';
  showAccessDeniedModal = false;
  submitting = false;
  
  rules: ControlRule[] = [];

  // Modal State
  showModal = false;
  isEditMode = false;
  currentRuleId: string | null = null;
  
  // Delete Confirmation
  showDeleteConfirmModal = false;
  deleteRuleId: string | null = null;
  deleteRuleName: string = '';
  
  formData = {
    nomRegle: '',
    description: '',
    typeRegle: '',
    active: true,
    seuil: 0,
    profilClientId: '',
    periodeReference: '7d',
    action: 'surveiller'
  };

  availableActions = [
    { value: 'bloquer', label: 'Bloquer' },
    { value: 'surveiller', label: 'Surveiller' }
  ];

  predefinedRuleTypes: string[] = ['montant', 'frequence', 'pays_risque'];
  ruleTypes: string[] = [];
  profils: Array<{ id: string; label: string }> = [];

  constructor(
    private controlRulesService: ControlRulesService, 
    private authService: AuthService, 
    private config: ConfigService
  ) {}

  ngOnInit(): void {
    if (typeof window === 'undefined') return;
    this.isAdmin = this.authService.isAdmin();
    this.loadProfilsClient();
    this.loadRules();
  }

  loadProfilsClient(): void {
    this.controlRulesService.getProfilsClient().subscribe({
      next: (data: ProfilClient[]) => {
        this.profils = data.map(p => ({ id: p.id, label: p.nomProfil }));
      },
      error: () => this.profils = []
    });
  }

  loadRules(): void {
    this.loading = true;
    this.controlRulesService.getRules().subscribe({
      next: (data: any) => {
        this.rules = Array.isArray(data) ? data : (data.rules || []);
        this.ruleTypes = [...this.predefinedRuleTypes].sort();
        this.loading = false;
      },
      error: () => {
        this.error = 'Impossible de charger les règles';
        this.loading = false;
      }
    });
  }

  getRuleIcon(type: string): string {
    switch (type) {
      case 'montant': return 'monetization_on';
      case 'frequence': return 'history_toggle_off';
      case 'pays_risque': return 'public';
      default: return 'gavel';
    }
  }

  onRuleTypeChange(): void {
    if (this.formData.typeRegle === 'montant') this.formData.periodeReference = '7d';
    else if (this.formData.typeRegle === 'frequence') this.formData.periodeReference = '24h';
    else this.formData.periodeReference = '';
  }

  saveRule(): void {
    if (this.submitting) return;
    this.submitting = true;
    const payload = { ...this.formData };

    const obs = this.isEditMode && this.currentRuleId 
      ? this.controlRulesService.updateRule(this.currentRuleId, payload)
      : this.controlRulesService.createRule(payload);

    obs.subscribe({
      next: () => {
        this.loadRules();
        this.closeModal();
        this.submitting = false;
      },
      error: () => {
        this.error = 'Erreur lors de l\'enregistrement';
        this.submitting = false;
      }
    });
  }

  toggleRuleStatus(rule: ControlRule): void {
    if (!this.isAdmin) return this.showAccessDenied();
    this.submitting = true;
    const nextState = !rule.active;
    this.controlRulesService.toggleRuleStatus(rule.id, nextState ? 'Actif' : 'Inactif').subscribe({
      next: () => { rule.active = nextState; this.submitting = false; },
      error: () => this.submitting = false
    });
  }

  openEditRuleModal(rule: ControlRule): void {
    this.isEditMode = true;
    this.currentRuleId = rule.id;
    this.formData = { ...rule } as any;
    this.showModal = true;
  }

  addRule(): void {
    if (!this.isAdmin) return this.showAccessDenied();
    this.isEditMode = false;
    this.resetForm();
    this.showModal = true;
  }

  deleteRule(id: string): void {
    if (!this.isAdmin) return this.showAccessDenied();
    const rule = this.rules.find(r => r.id === id);
    this.deleteRuleId = id;
    this.deleteRuleName = rule?.nomRegle || '';
    this.showDeleteConfirmModal = true;
  }

  confirmDeleteRule(): void {
    if (!this.deleteRuleId) return;
    this.controlRulesService.deleteRule(this.deleteRuleId).subscribe({
      next: () => { this.loadRules(); this.closeDeleteConfirmModal(); },
      error: () => this.submitting = false
    });
  }

  resetForm() { this.formData = { nomRegle: '', description: '', typeRegle: 'frequence', active: true, seuil: 0, profilClientId: '', periodeReference: '7d', action: 'surveiller' }; }
  closeModal() { this.showModal = false; this.error = ''; }
  closeDeleteConfirmModal() { this.showDeleteConfirmModal = false; }
  showAccessDenied() { this.showAccessDeniedModal = true; }
  closeAccessDeniedModal() { this.showAccessDeniedModal = false; }
  getProfilName(id?: string) { return this.profils.find(p => p.id === id)?.label || 'Tous'; }
  getPeriodLabel(p?: string) { return p === '24h' ? '24h' : p === '7d' ? '7 jours' : p === '30d' ? '30 jours' : p; }
  editRule(rule: ControlRule) { this.openEditRuleModal(rule); }
}