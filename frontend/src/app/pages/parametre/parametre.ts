import { Component, OnInit, LOCALE_ID } from '@angular/core';
import { CommonModule, registerLocaleData } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ProfileService } from '../../services/profile.service';
import { AuthStateService } from '../../services/auth-state.service';
import { Router } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { HttpErrorResponse } from '@angular/common/http';
import localeFr from '@angular/common/locales/fr';

registerLocaleData(localeFr);

@Component({
  selector: 'app-parametre',
  standalone: true,
  imports: [CommonModule, FormsModule, MatIconModule],
  providers: [{ provide: LOCALE_ID, useValue: 'fr-FR' }],
  templateUrl: './parametre.html',
  styleUrls: ['./parametre.scss']
})
export class ParametreComponent implements OnInit {
  activeTab: 'profile' | 'security' = 'profile';

  profile: any = null;
  saving = false;
  message = '';
  messageType: 'success' | 'error' | '' = '';
  oldPassword = '';
  newPassword = '';
  confirmPassword = '';
  showOldPassword = false;
  showNewPassword = false;
  showConfirmPassword = false;
  private originalProfile: string | null = null;

  constructor(
    private readonly profileService: ProfileService,
    private readonly authState: AuthStateService,
    private readonly router: Router
  ) {}

  ngOnInit(): void {
    this.loadProfile();
  }

  loadProfile() {
    this.profileService.getProfile().subscribe({
      next: (p: any) => { this.profile = p || null; this.setOriginalSnapshot(); },
      error: (_err: any) => { this.profile = null; }
    });
  }

  // track original profile for cancel/detect changes
  private setOriginalSnapshot() {
    if (!this.profile) {
      this.originalProfile = null;
      return;
    }
    this.originalProfile = JSON.stringify({
      nomUtilisateur: this.profile.nomUtilisateur || this.profile.nom_utilisateur,
      email: this.profile.email
    });
  }

  // Helper to show role label in French
  roleLabel(role: string | undefined | null): string {
    if (!role) return '-';
    const r = role.toLowerCase();
    if (r === 'admin') return 'Administrateur';
    if (r === 'analyst' || r === 'analyste') return 'Analyste';
    // fallback: capitalise
    return r.charAt(0).toUpperCase() + r.slice(1);
  }

  get isDirty(): boolean {
    if (!this.profile) return false;
    const current = JSON.stringify({
      nomUtilisateur: this.profile.nomUtilisateur || this.profile.nom_utilisateur,
      email: this.profile.email
    });
    return this.originalProfile !== current;
  }

  refreshProfile() {
    this.clearMessage();
    this.loadProfile();
  }

  setTab(t: 'profile' | 'security') { this.activeTab = t; this.clearMessage(); }
  clearMessage() { this.message = ''; this.messageType = ''; }

  toggleOldPassword() { this.showOldPassword = !this.showOldPassword; }
  toggleNewPassword() { this.showNewPassword = !this.showNewPassword; }
  toggleConfirmPassword() { this.showConfirmPassword = !this.showConfirmPassword; }

  // Save profile edits (name/email)
  saveProfileEdits() {
    if (!this.profile) return;
    this.saving = true;
    this.clearMessage();
    const payload: any = {
      nomUtilisateur: this.profile.nomUtilisateur || this.profile.nom_utilisateur,
      email: this.profile.email
    };
    this.profileService.updateProfile(payload).subscribe({
      next: (updated: any) => {
        this.saving = false;
        this.profile = updated || this.profile;
        this.message = 'Profil mis à jour.';
        this.messageType = 'success';
        this.setOriginalSnapshot();
      },
      error: (err: any) => {
        this.saving = false;
        this.message = this.getProfileEditErrorMessage(err);
        this.messageType = 'error';
      }
    });
  }

  // Get user-friendly error message for profile edit
  private getProfileEditErrorMessage(error: any): string {
    if (error instanceof HttpErrorResponse) {
      switch (error.status) {
        case 400:
          return 'Les données fournies sont invalides.';
        case 409:
          return 'Cet email est déjà utilisé par un autre utilisateur.';
        case 0:
          return 'Le serveur est indisponible ou non démarré.';
        case 500:
        case 502:
        case 503:
          return 'Erreur côté serveur. Veuillez réessayer plus tard.';
        default:
          return error.error?.message || 'Une erreur est survenue lors de la mise à jour du profil.';
      }
    }
    return 'Une erreur est survenue lors de la mise à jour du profil.';
  }

  // Change password
  changePassword(oldPassword: string, newPassword: string, confirm: string) {
    this.clearMessage();
    if (!oldPassword) { this.message = 'Veuillez saisir votre ancien mot de passe.'; this.messageType = 'error'; return; }
    if (!newPassword || newPassword.length < 6) { this.message = 'Le nouveau mot de passe doit contenir au moins 6 caractères.'; this.messageType = 'error'; return; }
    if (newPassword !== confirm) { this.message = 'La confirmation ne correspond pas.'; this.messageType = 'error'; return; }
    this.saving = true;
    this.profileService.updateProfile({ oldPassword: oldPassword, password: newPassword }).subscribe({
      next: () => { 
        this.saving = false; 
        this.message = 'Mot de passe mis à jour avec succès.'; 
        this.messageType = 'success'; 
        this.oldPassword = ''; 
        this.newPassword = ''; 
        this.confirmPassword = ''; 
      },
      error: (err: any) => { 
        this.saving = false; 
        console.error('Password change error:', err);
        this.message = this.getPasswordChangeErrorMessage(err); 
        this.messageType = 'error'; 
      }
    });
  }

  // Get user-friendly error message for password change
  private getPasswordChangeErrorMessage(error: any): string {
    if (error instanceof HttpErrorResponse) {
      switch (error.status) {
        case 401:
          return 'L\'ancien mot de passe est incorrect.';
        case 400:
          return 'Les données fournies sont invalides.';
        case 0:
          return 'Le serveur backend est indisponible ou non démarré.';
        case 500:
        case 502:
        case 503:
          return 'Erreur côté serveur. Veuillez réessayer plus tard.';
        default:
          return error.error?.message || 'Une erreur est survenue lors du changement de mot de passe.';
      }
    }
    return 'Une erreur est survenue lors du changement de mot de passe.';
  }

  // Logout handler used by the left card
  onLogout() {
    this.authState.logout();
    if (globalThis.window?.localStorage) {
      globalThis.window.localStorage.removeItem('token');
    }
    this.router.navigate(['/login']);
  }
}

