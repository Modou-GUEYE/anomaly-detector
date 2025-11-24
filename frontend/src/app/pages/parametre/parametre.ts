import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ProfileService } from '../../services/profile.service';
import { AuthStateService } from '../../services/auth-state.service';
import { Router } from '@angular/router';

@Component({
  selector: 'app-parametre',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './parametre.html',
  styleUrls: ['./parametre.scss']
})
export class ParametreComponent implements OnInit {
  activeTab: 'profile' | 'security' | 'notifications' = 'profile';

  profile: any = null;
  saving = false;
  message = '';
  messageType: 'success' | 'error' | '' = '';
  newPassword = '';
  confirmPassword = '';
  private originalProfile: string | null = null;

  constructor(private profileService: ProfileService, private authState: AuthStateService, private router: Router) {}

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
    try {
      this.originalProfile = this.profile ? JSON.stringify({ nomUtilisateur: this.profile.nomUtilisateur || this.profile.nom_utilisateur, email: this.profile.email }) : null;
    } catch (e) { this.originalProfile = null; }
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
    try {
      const current = JSON.stringify({ nomUtilisateur: this.profile.nomUtilisateur || this.profile.nom_utilisateur, email: this.profile.email });
      return this.originalProfile !== current;
    } catch (e) { return false; }
  }

  refreshProfile() {
    this.clearMessage();
    this.loadProfile();
  }

  setTab(t: 'profile' | 'security' | 'notifications') { this.activeTab = t; this.clearMessage(); }
  clearMessage() { this.message = ''; this.messageType = ''; }

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
        this.message = err?.error?.error || 'Erreur lors de la mise à jour du profil';
        this.messageType = 'error';
      }
    });
  }

  // Change password
  changePassword(newPassword: string, confirm: string) {
    this.clearMessage();
    if (!newPassword || newPassword.length < 6) { this.message = 'Le mot de passe doit contenir au moins 6 caractères.'; this.messageType = 'error'; return; }
    if (newPassword !== confirm) { this.message = 'La confirmation ne correspond pas.'; this.messageType = 'error'; return; }
    this.saving = true;
    this.profileService.updateProfile({ password: newPassword }).subscribe({
      next: () => { this.saving = false; this.message = 'Mot de passe mis à jour.'; this.messageType = 'success'; },
      error: (err: any) => { this.saving = false; this.message = err?.error?.error || 'Erreur lors de la mise à jour du mot de passe'; this.messageType = 'error'; }
    });
  }

  // Save notifications preferences
  saveNotifications() {
    if (!this.profile) return;
    this.clearMessage();
    this.saving = true;
    const payload: any = { notificationsEnabled: !!this.profile.notificationsEnabled };
    this.profileService.updateProfile(payload).subscribe({
      next: (updated: any) => {
        this.saving = false;
        this.profile = { ...this.profile, ...(updated || {}) };
        this.message = 'Préférences mises à jour.';
        this.messageType = 'success';
        this.setOriginalSnapshot();
      },
      error: (err: any) => {
        this.saving = false;
        this.message = err?.error?.error || 'Erreur lors de la sauvegarde des préférences';
        this.messageType = 'error';
      }
    });
  }

  // Logout handler used by the left card
  onLogout() {
    try {
      this.authState.logout();
    } catch (e) {
      try { localStorage.removeItem('token'); } catch (e) {}
      this.router.navigate(['/login']);
    }
  }
}

