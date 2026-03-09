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

  formErrors: { [key: string]: string } = {
    nomRegle: '',
    description: '',
    typeRegle: '',
    seuil: '',
    periodeReference: '',
    action: ''
  };

  touchedFields: { [key: string]: boolean } = {
    nomRegle: false,
    description: false,
    typeRegle: false,
    seuil: false,
    periodeReference: false,
    action: false
  };

  constructor(
    private readonly controlRulesService: ControlRulesService,
    private readonly authService: AuthService,
    private readonly config: ConfigService
  ) {}

  ngOnInit(): void {
    if (globalThis.window === undefined) return;
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
        this.ruleTypes = [...this.predefinedRuleTypes].sort((left, right) => left.localeCompare(right));
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

    this.validateField('typeRegle');
    this.validateField('seuil');
    this.validateField('periodeReference');
  }

  saveRule(): void {
    if (this.submitting) return;

    this.error = '';
    if (!this.validateForm(true)) {
      return;
    }

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
    this.formData = {
      nomRegle: rule.nomRegle || '',
      description: rule.description || '',
      typeRegle: rule.typeRegle || 'frequence',
      active: rule.active,
      seuil: rule.seuil || 0,
      profilClientId: rule.profilClientId || '',
      periodeReference: rule.periodeReference || '7d',
      action: rule.action || 'surveiller'
    };
    this.resetValidation();
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

  resetForm() {
    this.formData = { nomRegle: '', description: '', typeRegle: 'frequence', active: true, seuil: 0, profilClientId: '', periodeReference: '7d', action: 'surveiller' };
    this.resetValidation();
  }

  closeModal() {
    this.showModal = false;
    this.error = '';
    this.resetValidation();
  }
  closeDeleteConfirmModal() { this.showDeleteConfirmModal = false; }
  showAccessDenied() { this.showAccessDeniedModal = true; }
  closeAccessDeniedModal() { this.showAccessDeniedModal = false; }
  getProfilName(id?: string) { return this.profils.find(p => p.id === id)?.label || 'Tous'; }
  getPeriodLabel(p?: string) {
    if (p === '24h') return '24h';
    if (p === '7d') return '7 jours';
    if (p === '30d') return '30 jours';
    return p;
  }
  editRule(rule: ControlRule) { this.openEditRuleModal(rule); }

  markFieldTouched(field: string): void {
    this.touchedFields[field] = true;
    this.validateField(field);
  }

  onFieldChange(field: string): void {
    if (this.touchedFields[field]) {
      this.validateField(field);
    }
  }

  shouldShowFieldError(field: string): boolean {
    return !!this.formErrors[field] && !!this.touchedFields[field];
  }

  private validateForm(markTouched = false): boolean {
    if (markTouched) {
      Object.keys(this.touchedFields).forEach((key) => {
        this.touchedFields[key] = true;
      });
    }

    this.validateField('nomRegle');
    this.validateField('description');
    this.validateField('typeRegle');
    this.validateField('seuil');
    this.validateField('periodeReference');
    this.validateField('action');

    return !Object.values(this.formErrors).some((value) => !!value);
  }

  private validateField(field: string): void {
    const validators: Record<string, () => string> = {
      nomRegle: () => this.getNomRegleError(),
      description: () => this.getDescriptionError(),
      typeRegle: () => this.getTypeRegleError(),
      seuil: () => this.getSeuilError(),
      periodeReference: () => this.getPeriodeReferenceError(),
      action: () => this.getActionError()
    };

    const validate = validators[field];
    if (validate) {
      this.formErrors[field] = validate();
    }
  }

  private getNomRegleError(): string {
    const nomRegle = this.formData.nomRegle.trim();
    if (!nomRegle) {
      return 'Le nom de la règle est obligatoire.';
    }
    if (nomRegle.length < 3) {
      return 'Le nom de la règle doit contenir au moins 3 caractères.';
    }
    return '';
  }

  private getDescriptionError(): string {
    const description = this.formData.description.trim();
    if (!description) {
      return 'La description est obligatoire.';
    }
    if (description.length < 10) {
      return 'La description doit contenir au moins 10 caractères.';
    }
    return '';
  }

  private getTypeRegleError(): string {
    return this.formData.typeRegle ? '' : 'Le type de règle est obligatoire.';
  }

  private getSeuilError(): string {
    if ((this.formData.typeRegle === 'montant' || this.formData.typeRegle === 'frequence') && Number(this.formData.seuil) <= 0) {
      return 'La valeur seuil doit être supérieure à 0.';
    }
    return '';
  }

  private getPeriodeReferenceError(): string {
    if (this.formData.typeRegle !== 'pays_risque' && !this.formData.periodeReference) {
      return 'La période est obligatoire.';
    }
    return '';
  }

  private getActionError(): string {
    return this.formData.action ? '' : 'L\'action est obligatoire.';
  }

  private resetValidation(): void {
    this.formErrors = {
      nomRegle: '',
      description: '',
      typeRegle: '',
      seuil: '',
      periodeReference: '',
      action: ''
    };

    this.touchedFields = {
      nomRegle: false,
      description: false,
      typeRegle: false,
      seuil: false,
      periodeReference: false,
      action: false
    };
  }
}