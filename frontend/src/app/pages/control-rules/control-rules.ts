import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ControlRulesService, ControlRule, ProfilClient } from '../../services/control-rules.service';
import { AuthService } from '../../services/auth.service';

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
  isAdmin = false;
  adminOnlyMessage = 'Cette action est réservée aux administrateurs';
  showAccessDeniedModal = false;
  
  rules: ControlRule[] = [];

  // Modal
  showModal = false;
  isEditMode = false;
  currentRuleId: string | null = null;
  
  // Formulaire
  formData = {
    nomRegle: '',
    description: '',
    typeRegle: '',
    active: true,
    seuil: 0,
    profilClientId: '',
    periodeReference: '7d' // Valeur par défaut
  };

  // Types de règles : utiliser les types backend directement
  predefinedRuleTypes: string[] = [
    'montant',
    'frequence',
    'pays_risque'
  ];
  
  ruleTypes: string[] = [];  // Fusion des types prédéfinis et existants

  // Profils clients - chargés dynamiquement de l'API
  profils: Array<{ id: string; label: string }> = [];

  constructor(private controlRulesService: ControlRulesService, private authService: AuthService) {}

  ngOnInit(): void {
    if (typeof window === 'undefined') {
      return;
    }
    
    // Vérifier si l'utilisateur est admin
    this.isAdmin = this.authService.isAdmin();
    
    this.loadProfilsClient();
    this.loadRules();
  }

  /**
   * Charge les profils clients depuis l'API
   */
  loadProfilsClient(): void {
    this.controlRulesService.getProfilsClient().subscribe({
      next: (data: ProfilClient[]) => {
        // Transforme les profils du backend en format utilisable
        this.profils = data.map(profil => ({
          id: profil.id,
          label: profil.nomProfil
        }));
      },
      error: (err) => {
        // En cas d'erreur, laisser la liste vide
        // L'utilisateur pourra créer une règle sans profil spécifique
        this.profils = [];
        console.warn('Impossible de charger les profils clients:', err);
      }
    });
  }

  /**
   * Utilise uniquement les types prédéfinis
   */
  extractRuleTypes(): void {
    this.ruleTypes = [...this.predefinedRuleTypes].sort();
  }

  // Appelée quand le type de règle change dans le formulaire
  onRuleTypeChange(): void {
    // Si on change vers montant, initialiser periodeReference à 7d
    if (this.formData.typeRegle === 'montant' && !this.formData.periodeReference) {
      this.formData.periodeReference = '7d';
    } 
    // Si on change vers frequence, initialiser periodeReference à 24h
    else if (this.formData.typeRegle === 'frequence' && !this.formData.periodeReference) {
      this.formData.periodeReference = '24h';
    } 
    // Pour pays_risque et autres, vider periodeReference
    else if (this.formData.typeRegle !== 'montant' && this.formData.typeRegle !== 'frequence') {
      this.formData.periodeReference = '';
    }
  }

  loadRules(): void {
    this.loading = true;
    this.error = '';
    
    this.controlRulesService.getRules().subscribe({
      next: (data: any) => {
        // Assure que data est un tableau
        this.rules = Array.isArray(data) ? data : (data.rules || []);
        // Enlever les doublons par ID
        this.rules = this.rules.filter((rule, index, self) =>
          index === self.findIndex((r) => r.id === rule.id)
        );
        // Extraire les types de règles uniques
        this.extractRuleTypes();
        // Debug: afficher les données reçues
        this.rules.forEach(rule => {
          if (rule.profilClientId) {
          }
        });
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Impossible de charger les règles de contrôle';
        this.loading = false;
      }
    });
  }

  openAddRuleModal(): void {
    if (!this.isAdmin) {
      this.showAccessDenied();
      return;
    }
    this.isEditMode = false;
    this.currentRuleId = null;
    this.resetForm();
    this.showModal = true;
  }

  openEditRuleModal(rule: ControlRule): void {
    this.isEditMode = true;
    this.currentRuleId = rule.id;
    this.formData = {
      nomRegle: rule.nomRegle,
      description: rule.description,
      typeRegle: rule.typeRegle,
      active: rule.active,
      seuil: rule.seuil || 0,
      profilClientId: rule.profilClientId || '',
      // periodeReference seulement pour montant et frequence
      periodeReference: (rule.typeRegle === 'montant' || rule.typeRegle === 'frequence') ? (rule.periodeReference || '7d') : ''
    };
    this.showModal = true;
  }

  closeModal(): void {
    this.showModal = false;
    this.resetForm();
  }

  resetForm(): void {
    this.formData = {
      nomRegle: '',
      description: '',
      typeRegle: '',
      active: true,
      seuil: 0,
      profilClientId: '',
      periodeReference: '' // Sera défini selon le type
    };
  }

  saveRule(): void {
    if (!this.formData.nomRegle?.trim() || !this.formData.description?.trim() || !this.formData.typeRegle) {
      alert('Veuillez remplir tous les champs obligatoires');
      return;
    }

    const payload: Partial<ControlRule> = {
      nomRegle: this.formData.nomRegle.trim(),
      description: this.formData.description.trim(),
      typeRegle: this.formData.typeRegle,
      active: this.formData.active,
      seuil: this.formData.seuil > 0 ? this.formData.seuil : undefined,
      profilClientId: this.formData.profilClientId || undefined
    };

    // periodeReference est valide pour montant et frequence
    if (this.formData.typeRegle === 'montant' || this.formData.typeRegle === 'frequence') {
      payload.periodeReference = this.formData.periodeReference;
    } else {
      // Pour les autres types (pays_risque), ne pas envoyer periodeReference
      payload.periodeReference = undefined;
    }

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
    if (!this.isAdmin) {
      this.showAccessDenied();
      return;
    }
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
    if (!this.isAdmin) {
      this.showAccessDenied();
      return;
    }
    const newActive = !rule.active;
    
    this.controlRulesService.toggleRuleStatus(rule.id, newActive ? 'Actif' : 'Inactif').subscribe({
      next: () => {
        rule.active = newActive;
      },
      error: (err) => {
        this.error = 'Erreur lors de la modification du statut';
      }
    });
  }

  editRule(rule: ControlRule): void {
    if (!this.isAdmin) {
      this.showAccessDenied();
      return;
    }
    this.openEditRuleModal(rule);
  }

  addRule(): void {
    this.openAddRuleModal();
  }

  getProfilName(profilId?: string): string {
    if (!profilId) return '-';
    const profil = this.profils.find(p => p.id === profilId);
    return profil ? profil.label : '-';
  }

  showAccessDenied(): void {
    this.showAccessDeniedModal = true;
  }

  closeAccessDeniedModal(): void {
    this.showAccessDeniedModal = false;
  }
}
