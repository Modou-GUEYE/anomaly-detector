import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { UsersService } from '../../services/users.service';
import { ConfigService } from '../../services/config.service';
import { Router } from '@angular/router';

@Component({
  selector: 'app-signin',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule],
  templateUrl: './signin.html',
  styleUrls: ['./signin.scss']
})
export class SigninComponent {
  prenom = '';
  nom = '';
  email = '';
  password = '';
  role = '';
  loading = false;
  success = false;
  message = '';
  showPassword = false;
  acceptedPolicy = false;

  roles: Array<{ value: string; label: string }> = [];

  constructor(private usersService: UsersService, private config: ConfigService, private router: Router) {
    // initialize role options from runtime config (defensive)
    let adminRole = 'admin';
    let analystRole = 'analyst';
    try { const r = this.config.getRoleName('admin'); if (r) adminRole = r; } catch (e) {}
    try { const r = this.config.getRoleName('analyst'); if (r) analystRole = r; } catch (e) {}

    this.roles = [
      { value: analystRole, label: 'Analyste' },
      { value: adminRole, label: 'Administrateur' }
    ];

    if (!this.role) this.role = analystRole;
  }

  togglePassword() { this.showPassword = !this.showPassword; }

  submit() {
    this.message = '';
    this.success = false;

    const nomUtilisateur = `${(this.prenom || '').trim()} ${(this.nom || '').trim()}`.trim();

    if (!nomUtilisateur || !this.email || !this.password || !this.role) {
      this.message = 'Veuillez remplir tous les champs requis.';
      return;
    }
    if (!this.acceptedPolicy) {
      this.message = 'Vous devez accepter la politique de confidentialité pour vous inscrire.';
      return;
    }
    if (this.password.length < 6) {
      this.message = 'Le mot de passe doit contenir au moins 6 caractères.';
      return;
    }

    this.loading = true;
    const payload = {
      nomUtilisateur: nomUtilisateur,
      email: this.email,
      password: this.password,
      role: this.role,
      actif: false // comptes créés via l'inscription sont inactifs par défaut
    };

  this.usersService.signup(payload).subscribe({
      next: (res) => {
        this.loading = false;
        this.success = true;
        this.message = "Compte créé avec succès. Un administrateur doit l'activer avant la première connexion.";
        // clear form
        this.prenom = '';
        this.nom = '';
        this.email = '';
        this.password = '';
        this.acceptedPolicy = false;
      },
      error: (err) => {
        this.loading = false;
        this.message = err?.error?.error || err?.message || 'Erreur lors de la création du compte. Veuillez contacter un administrateur.';
      }
    });
  }

  goToLogin() {
    try { this.router.navigate(['/login']); } catch (e) { window.location.href = '/login'; }
  }
}
