import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { UsersService } from '../../services/users.service';
import { OnInit } from '@angular/core';
import { ConfigService } from '../../services/config.service';

@Component({
  selector: 'app-create-user',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './create-user.html',
  styleUrls: ['./create-user.scss']
})
export class CreateUserComponent implements OnInit {
  nomUtilisateur = '';
  email = '';
  password = '';
  role: string = '';
  actif = true;
  loading = false;
  error = '';
  submitted = false;
  showPassword = false;
  users: any[] = [];
  // Filters
  filterText = '';
  filterRole: string = 'all';
  filterActif: string = 'all';
  showModal = false;
  // Track per-user action loading states (activate/delete)
  actionLoading: { [id: string]: boolean } = {};
  // Delete confirmation modal
  showDeleteModal = false;
  userToDelete: any = null;
  deleteError = '';
  deleteBlocked = false;
  // feedback is shown inline (modal messages) — toasts removed

  roles: Array<{ value: string; label: string }> = [];

  constructor(private usersService: UsersService, public router: Router, private config: ConfigService) {}

  ngOnInit(): void {
    // initialize role defaults from runtime config (defensive)
    let adminRole = 'admin';
    let analystRole = 'analyst';
    try { const r = this.config.getRoleName('admin'); if (r) adminRole = r; } catch (e) {}
    try { const r = this.config.getRoleName('analyst'); if (r) analystRole = r; } catch (e) {}

    this.roles = [
      { value: adminRole, label: 'Administrateur' },
      { value: analystRole, label: 'Analyste' }
    ];

    // valeur par défaut
    if (!this.role) this.role = analystRole;
    this.loadUsers();
  }

  submit() {
    this.submitted = true;
    this.error = '';
    if (!this.nomUtilisateur || !this.email || !this.password || !this.role) {
      this.error = 'Veuillez remplir tous les champs requis.';
      return;
    }
    if (this.password.length < 6) {
      this.error = 'Le mot de passe doit contenir au moins 6 caractères.';
      return;
    }

    this.loading = true;
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
        // Fermer le modal et recharger la liste
        this.resetForm();
        this.showModal = false;
        this.loadUsers();
      },
      error: (err) => {
        this.loading = false;
        this.error = err?.error?.error || err?.message || 'Erreur lors de la création de l\'utilisateur';
      }
    });
  }

  toggleShowPassword() {
    this.showPassword = !this.showPassword;
  }

  passwordStrength(): 'weak' | 'medium' | 'strong' {
    const p = this.password || '';
    let score = 0;
    if (p.length >= 6) score++;
    if (/[A-Z]/.test(p)) score++;
    if (/[0-9]/.test(p)) score++;
    if (/[^A-Za-z0-9]/.test(p)) score++;
    if (score <= 1) return 'weak';
    if (score === 2) return 'medium';
    return 'strong';
  }

  resetForm() {
    this.nomUtilisateur = '';
    this.email = '';
    this.password = '';
    try { this.role = this.config.getRoleName('analyst') || 'analyst'; } catch (e) { this.role = 'analyst'; }
    this.actif = true;
    this.error = '';
  }

  loadUsers() {
    this.usersService.getUsers().subscribe({
      next: (data) => {
        this.users = Array.isArray(data) ? data : [];
      },
      error: (err) => {
        // error handled silently; consider showing a toast if needed
      }
    });
  }

  get filteredUsers() {
    const text = (this.filterText || '').toLowerCase().trim();
    return (this.users || []).filter((u: any) => {
      // role filter
      if (this.filterRole && this.filterRole !== 'all' && String(u.role) !== String(this.filterRole)) return false;
      // actif filter
      if (this.filterActif === 'actif' && !u.actif) return false;
      if (this.filterActif === 'inactif' && u.actif) return false;
      // text search on name or email
      if (text) {
        const name = ((u.nomUtilisateur || u.nom_utilisateur) || '').toString().toLowerCase();
        const email = (u.email || '').toString().toLowerCase();
        return name.includes(text) || email.includes(text);
      }
      return true;
    });
  }

  clearFilters() {
    this.filterText = '';
    this.filterRole = 'all';
    this.filterActif = 'all';
  }

  toggleActive(user: any) {
    const newActif = !user.actif;
    this.actionLoading[user.id] = true;
    this.usersService.updateUser(user.id, { actif: newActif }).subscribe({
      next: (updated) => {
        user.actif = updated.actif;
        this.actionLoading[user.id] = false;
      },
      error: (err) => {
        // activation failed; consider showing a toast if needed
        this.actionLoading[user.id] = false;
      }
    });
  }

  // open delete confirmation modal
  deleteUserConfirm(user: any) {
    this.userToDelete = user;
    this.deleteError = '';
    this.deleteBlocked = false;
    this.showDeleteModal = true;
  }

  // confirm deletion
  confirmDelete() {
    if (!this.userToDelete) return;
    const id = this.userToDelete.id;
    this.actionLoading[id] = true;
    this.deleteError = '';
    this.deleteBlocked = false;
    this.usersService.deleteUser(id).subscribe({
      next: () => {
  this.users = this.users.filter(u => u.id !== id);
  this.actionLoading[id] = false;
  this.deleteError = '';
  this.closeDeleteModal();
      },
      error: (err) => {
  // deletion error handled by parsing and user feedback
        // Try to extract a meaningful error payload from several possible shapes
        let raw: string = '';
        try {
          if (err && err.error) {
            if (typeof err.error === 'string') raw = err.error;
            else if (err.error.error) raw = err.error.error;
            else raw = JSON.stringify(err.error);
          }
        } catch (e) {
          raw = '';
        }
        if (!raw) raw = err?.message || 'Erreur lors de la suppression';

        // Base parsed result on message probes
        const parsed = this.parseBackendError(raw);

        // Aggressive fallback: if API returned 500 (or 409) for a DELETE on users, assume FK/constraint
        try {
          const url = (err && (err.url || err.error?.url)) ? String(err.url || err.error?.url) : '';
          if (!parsed.blocked && err && (err.status === 500 || err.status === 409)) {
            if (url.includes('/api/v1/users/') || /foreign key|violates|constraint|pq:|cannot delete|referenc/.test((raw || '').toLowerCase())) {
              parsed.blocked = true;
              parsed.message = "Impossible de supprimer l'utilisateur : il est référencé par d'autres données (logs, audit, ...). Vous pouvez le désactiver à la place.";
            }
          }
        } catch (e) {
          // ignore
        }

        this.deleteError = parsed.message;
        this.deleteBlocked = parsed.blocked;
        this.actionLoading[id] = false;
        // keep the modal open so the user can see the error (shown in deleteError)
      }
    });
  }

  private formatBackendError(raw: string): string {
    // kept for backward compatibility; prefer parseBackendError
    return (raw || '').toString();
  }

  private parseBackendError(raw: string): { message: string; blocked: boolean } {
    const s = (raw || '').toString();
    const lower = s.toLowerCase();
    const isFK = lower.includes('violates foreign key') || lower.includes('violates') || lower.includes('foreign key') || lower.includes('constraint') || lower.includes('référence') || lower.includes('reference') || lower.includes('pq:') || lower.includes('cannot delete') || lower.includes('violates');
    if (isFK) {
      return {
        blocked: true,
        message: "Impossible de supprimer l'utilisateur : il est référencé par d'autres données (logs, audit, ...). Vous pouvez le désactiver à la place."
      };
    }
    // handle common auth/permission errors
    if (lower.includes('permission') || lower.includes('forbidden') || lower.includes('unauthorized') || lower.includes('401') || lower.includes('403')) {
      return { blocked: false, message: "Action interdite : vous n'avez pas la permission de supprimer cet utilisateur." };
    }
    // generic
    return { blocked: false, message: s };
  }

  // toasts removed: no-op

  openModal() {
    this.resetForm();
    this.showModal = true;
  }

  closeModal() {
    this.showModal = false;
  }

  closeDeleteModal() {
    this.showDeleteModal = false;
    this.userToDelete = null;
  }

  // Fallback: disable user instead of deleting when deletion is blocked by FK constraints
  disableInstead() {
    if (!this.userToDelete) return;
    const id = this.userToDelete.id;
    this.actionLoading[id] = true;
    this.usersService.updateUser(id, { actif: false }).subscribe({
      next: (updated) => {
        // update local user
        const idx = this.users.findIndex(u => u.id === id);
        if (idx !== -1) this.users[idx].actif = updated.actif;
        this.actionLoading[id] = false;
        this.deleteError = '';
        this.closeDeleteModal();
      },
      error: (err) => {
        // deactivation failed; surface a user-friendly message
        this.deleteError = err?.error?.error || err?.message || 'Erreur lors de la désactivation';
        this.actionLoading[id] = false;
      }
    });
  }
}
