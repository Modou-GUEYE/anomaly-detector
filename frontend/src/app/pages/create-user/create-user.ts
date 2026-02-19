import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { UsersService } from '../../services/users.service';
import { ConfigService } from '../../services/config.service';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-create-user',
  standalone: true,
  imports: [CommonModule, FormsModule, MatIconModule],
  templateUrl: './create-user.html',
  styleUrls: ['./create-user.scss']
})
export class CreateUserComponent implements OnInit {
  // Formulaire
  nomUtilisateur = '';
  email = '';
  password = '';
  role: string = '';
  actif = true;

  // États UI
  loading = false;
  showModal = false;
  showDeleteModal = false;
  error = '';
  editingUser: any = null; // Utilisateur en cours d'édition (null = création)
  
  // Gestion spécifique suppression
  deleteError = '';
  deleteBlocked = false;

  // Données
  users: any[] = [];
  roles: Array<{ value: string; label: string }> = [];
  userToDelete: any = null;
  actionLoading: { [id: string]: boolean } = {};

  // Filtres
  filterText = '';
  filterRole: string = 'all';
  filterActif: string = 'all';

  formErrors: { [key: string]: string } = {
    nomUtilisateur: '',
    email: '',
    password: '',
    role: ''
  };

  touchedFields: { [key: string]: boolean } = {
    nomUtilisateur: false,
    email: false,
    password: false,
    role: false
  };

  constructor(
    private usersService: UsersService,
    private config: ConfigService,
    public router: Router
  ) {}

  ngOnInit(): void {
    this.initRoles();
    this.loadUsers();
  }

  initRoles() {
    let adminRole = 'admin';
    let analystRole = 'analyst';
    try {
      adminRole = this.config.getRoleName('admin') || 'admin';
      analystRole = this.config.getRoleName('analyst') || 'analyst';
    } catch (e) {}

    this.roles = [
      { value: adminRole, label: 'Administrateur' },
      { value: analystRole, label: 'Analyste' }
    ];
    this.role = analystRole;
  }

  loadUsers() {
    this.usersService.getUsers().subscribe({
      next: (data) => this.users = Array.isArray(data) ? data : [],
      error: (err) => console.error('Erreur chargement users', err)
    });
  }

  get filteredUsers() {
    return this.users.filter(u => {
      const matchRole = this.filterRole === 'all' || u.role === this.filterRole;
      const matchStatus = this.filterActif === 'all' || 
                         (this.filterActif === 'actif' ? u.actif : !u.actif);
      const name = (u.nomUtilisateur || u.nom_utilisateur || '').toLowerCase();
      const email = (u.email || '').toLowerCase();
      const matchText = !this.filterText || 
                        name.includes(this.filterText.toLowerCase()) ||
                        email.includes(this.filterText.toLowerCase());
      return matchRole && matchStatus && matchText;
    });
  }

  submit() {
    this.error = '';
    if (!this.validateForm(true)) {
      return;
    }

    this.loading = true;

    // Mode création
    if (!this.editingUser) {
      const payload = { 
        nomUtilisateur: this.nomUtilisateur, 
        email: this.email, 
        password: this.password, 
        role: this.role, 
        actif: this.actif 
      };

      this.usersService.createUser(payload).subscribe({
        next: () => {
          this.loading = false;
          this.showModal = false;
          this.loadUsers();
        },
        error: (err: any) => {
          this.loading = false;
          this.error = err?.error?.error || err?.error?.message || 'Erreur technique';
        }
      });
    } 
    // Mode édition
    else {
      const id = this.editingUser.id || this.editingUser._id;
      const updates: any = {
        nomUtilisateur: this.nomUtilisateur, 
        email: this.email, 
        role: this.role, 
        actif: this.actif 
      };

      // Ajouter le password seulement s'il a été rempli
      if (this.password) {
        updates.password = this.password;
      }

      this.usersService.updateUser(id, updates).subscribe({
        next: () => {
          this.loading = false;
          this.showModal = false;
          this.loadUsers();
        },
        error: (err: any) => {
          this.loading = false;
          this.error = err?.error?.error || err?.error?.message || 'Erreur technique';
        }
      });
    }
  }

  toggleActive(user: any) {
    const id = user.id || user._id;
    this.actionLoading[id] = true;
    this.usersService.updateUser(id, { actif: !user.actif }).subscribe({
      next: (updated) => {
        user.actif = updated.actif;
        this.actionLoading[id] = false;
      },
      error: () => this.actionLoading[id] = false
    });
  }

  deleteUserConfirm(user: any) {
    this.userToDelete = user;
    this.deleteError = '';
    this.deleteBlocked = false;
    this.showDeleteModal = true;
  }

  confirmDelete() {
    if (!this.userToDelete) return;
    const id = this.userToDelete.id || this.userToDelete._id;
    this.actionLoading[id] = true;

    this.usersService.deleteUser(id).subscribe({
      next: () => {
        this.users = this.users.filter(u => (u.id || u._id) !== id);
        this.actionLoading[id] = false;
        this.showDeleteModal = false;
        this.userToDelete = null;
      },
      error: (err: any) => {
        this.actionLoading[id] = false;
        const rawError = err?.error?.error || err?.error?.message || err?.message || '';
        const parsed = this.parseBackendError(rawError, err.status);
        
        this.deleteError = parsed.message;
        this.deleteBlocked = parsed.blocked;
      }
    });
  }

  disableInstead() {
    if (!this.userToDelete) return;
    const id = this.userToDelete.id || this.userToDelete._id;
    this.actionLoading[id] = true;

    this.usersService.updateUser(id, { actif: false }).subscribe({
      next: () => {
        const user = this.users.find(u => (u.id || u._id) === id);
        if (user) user.actif = false;
        this.actionLoading[id] = false;
        this.showDeleteModal = false;
      },
      error: (err) => {
        this.actionLoading[id] = false;
        this.deleteError = "Impossible de désactiver l'utilisateur.";
      }
    });
  }

  private parseBackendError(raw: string, status: number): { message: string; blocked: boolean } {
    const lower = raw.toLowerCase();
    
    // Contrainte d'intégrité (Foreign Key)
    if (status === 500 || status === 409 || /foreign key|violates|constraint|référence|reference|cannot delete/.test(lower)) {
      return {
        blocked: true,
        message: "Impossible de supprimer cet utilisateur car il est lié à d'autres données (logs, audits). Souhaitez-vous le désactiver à la place ?"
      };
    }

    // Propre compte
    if (status === 403 || lower.includes('own account') || lower.includes('votre propre compte')) {
      return {
        blocked: false,
        message: "Action interdite : vous ne pouvez pas supprimer votre propre compte."
      };
    }

    return { blocked: false, message: raw || "Erreur lors de la suppression." };
  }

  openModal() {
    this.editingUser = null; // Mode création
    this.resetForm();
    this.showModal = true;
  }

  editUser(user: any) {
    this.editingUser = user; // Mode édition
    this.nomUtilisateur = user.nomUtilisateur || user.nom_utilisateur || '';
    this.email = user.email || '';
    this.password = ''; // Le password n'est pas affichable
    this.role = user.role || '';
    this.actif = user.actif !== undefined ? user.actif : true;
    this.error = '';
    this.resetValidation();
    this.showModal = true;
  }

  closeModal() {
    this.showModal = false;
    this.editingUser = null;
  }

  resetForm() {
    this.nomUtilisateur = '';
    this.email = '';
    this.password = '';
    this.role = this.roles?.[1]?.value || this.role;
    this.actif = true;
    this.error = '';
    this.resetValidation();
  }

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

    this.validateField('nomUtilisateur');
    this.validateField('email');
    this.validateField('password');
    this.validateField('role');

    return !Object.values(this.formErrors).some((value) => !!value);
  }

  private validateField(field: string): void {
    switch (field) {
      case 'nomUtilisateur':
        if (!this.nomUtilisateur.trim()) {
          this.formErrors['nomUtilisateur'] = 'Le nom d\'utilisateur est obligatoire.';
        } else if (this.nomUtilisateur.trim().length < 3) {
          this.formErrors['nomUtilisateur'] = 'Le nom d\'utilisateur doit contenir au moins 3 caractères.';
        } else {
          this.formErrors['nomUtilisateur'] = '';
        }
        break;

      case 'email': {
        const emailValue = this.email.trim();
        if (!emailValue) {
          this.formErrors['email'] = 'L\'adresse email est obligatoire.';
        } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailValue)) {
          this.formErrors['email'] = 'Le format de l\'email est invalide.';
        } else {
          this.formErrors['email'] = '';
        }
        break;
      }

      case 'password':
        if (!this.editingUser && !this.password) {
          this.formErrors['password'] = 'Le mot de passe est obligatoire.';
        } else if (this.password && this.password.length < 6) {
          this.formErrors['password'] = 'Le mot de passe doit contenir au moins 6 caractères.';
        } else {
          this.formErrors['password'] = '';
        }
        break;

      case 'role':
        if (!this.role) {
          this.formErrors['role'] = 'Le rôle est obligatoire.';
        } else {
          this.formErrors['role'] = '';
        }
        break;

      default:
        break;
    }
  }

  private resetValidation(): void {
    this.formErrors = {
      nomUtilisateur: '',
      email: '',
      password: '',
      role: ''
    };

    this.touchedFields = {
      nomUtilisateur: false,
      email: false,
      password: false,
      role: false
    };
  }
}