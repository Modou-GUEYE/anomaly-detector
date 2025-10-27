import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ControlRulesService, ControlRule } from '../../services/control-rules.service';

@Component({
  selector: 'app-control-rules',
  standalone: true,
  templateUrl: './control-rules.html',
  styleUrls: ['./control-rules.scss'],
  imports: [CommonModule, FormsModule]
})
export class ControlRulesComponent implements OnInit {
  loading = true;
  error = '';
  
  rules: ControlRule[] = [];

  // Modal
  showModal = false;
  isEditMode = false;
  currentRuleId: string | null = null;
  
  // Formulaire
  formData = {
    name: '',
    description: '',
    type: '',
    status: 'Actif' as 'Actif' | 'Inactif',
    threshold: 0
  };

  ruleTypes = [
    'Montant',
    'Fréquence',
    'Géographie',
    'Destinataire',
    'Autre'
  ];

  constructor(private controlRulesService: ControlRulesService) {}

  ngOnInit(): void {
    if (typeof window === 'undefined') {
      return;
    }
    this.loadRules();
  }

  loadRules(): void {
    this.loading = true;
    this.error = '';
    
    this.controlRulesService.getRules().subscribe({
      next: (data: any) => {
        this.rules = Array.isArray(data) ? data : (data.rules || []);
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Impossible de charger les règles de contrôle';
        this.loading = false;
      }
    });
  }

  openAddRuleModal(): void {
    this.isEditMode = false;
    this.currentRuleId = null;
    this.resetForm();
    this.showModal = true;
  }

  openEditRuleModal(rule: ControlRule): void {
    this.isEditMode = true;
    this.currentRuleId = rule.id;
    this.formData = {
      name: rule.name,
      description: rule.description,
      type: rule.type,
      status: rule.status,
      threshold: rule.threshold || 0
    };
    this.showModal = true;
  }

  closeModal(): void {
    this.showModal = false;
    this.resetForm();
  }

  resetForm(): void {
    this.formData = {
      name: '',
      description: '',
      type: '',
      status: 'Actif',
      threshold: 0
    };
  }

  saveRule(): void {
    if (!this.formData.name?.trim() || !this.formData.description?.trim() || !this.formData.type) {
      alert('Veuillez remplir tous les champs obligatoires');
      return;
    }

    const payload = {
      name: this.formData.name.trim(),
      description: this.formData.description.trim(),
      type: this.formData.type,
      status: this.formData.status,
      threshold: this.formData.threshold > 0 ? this.formData.threshold : null
    };

    if (this.isEditMode && this.currentRuleId) {
      // Mode édition
      this.controlRulesService.updateRule(this.currentRuleId, payload).subscribe({
        next: () => {
          this.loadRules();
          this.closeModal();
        },
        error: (err) => {
          this.error = 'Erreur lors de la mise à jour de la règle';
        }
      });
    } else {
      // Mode création
      this.controlRulesService.createRule(payload).subscribe({
        next: () => {
          this.loadRules();
          this.closeModal();
        },
        error: (err) => {
          this.error = 'Erreur lors de la création de la règle';
        }
      });
    }
  }

  deleteRule(id: string): void {
    if (!confirm('Êtes-vous sûr de vouloir supprimer cette règle ? Cette action est irréversible.')) {
      return;
    }

    this.controlRulesService.deleteRule(id).subscribe({
      next: () => {
        this.loadRules();
      },
      error: (err) => {
        this.error = 'Erreur lors de la suppression de la règle';
      }
    });
  }

  toggleRuleStatus(rule: ControlRule): void {
    const newStatus = rule.status === 'Actif' ? 'Inactif' : 'Actif';
    
    this.controlRulesService.toggleRuleStatus(rule.id, newStatus).subscribe({
      next: () => {
        rule.status = newStatus;
      },
      error: (err) => {
        this.error = 'Erreur lors de la modification du statut';
      }
    });
  }

  editRule(rule: ControlRule): void {
    this.openEditRuleModal(rule);
  }

  addRule(): void {
    this.openAddRuleModal();
  }
}
