
import { Component } from '@angular/core';
import { AuthService } from '../../services/auth.service';
import { AuthStateService } from '../../services/auth-state.service';
import { Router, RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { ConfigService } from '../../services/config.service';

@Component({
  selector: 'app-login',
  standalone: true,
  templateUrl: './login.html',
  styleUrls: ['./login.scss'],
  imports: [CommonModule, FormsModule, RouterModule],
})
export class LoginComponent {
  email: string = '';
  password: string = '';
  message: string = '';
  success: boolean = false;
  
  // role defaults to 'admin' for easier first login
  role: string = '';
  roleOptions: Array<{ value: string; label: string }> = [];
  showPassword: boolean = false;

  constructor(private authService: AuthService, private authState: AuthStateService, private router: Router, private config: ConfigService) {
    // Construire les options de rôle à partir du ConfigService avec des fallback simples
    let adminRole = 'admin';
    let analystRole = 'analyst';
    try { const r = this.config.getRoleName('admin'); if (r) adminRole = r; } catch (e) {}
    try { const r = this.config.getRoleName('analyst'); if (r) analystRole = r; } catch (e) {}

    this.roleOptions = [
      { value: adminRole, label: 'Admin' },
      { value: analystRole, label: 'Analyst' }
    ];

    // valeur par défaut
    if (!this.role) this.role = adminRole;
  }

  onSubmit() {
    // Handler du formulaire de connexion
    this.message = '';
    this.success = false;
    if (this.email && this.password && this.role) {
      this.message = 'Connexion...';
      this.authService.login(this.email, this.password, this.role).subscribe((ok) => {
        if (ok) {
          this.success = true;
          try { this.authState.isAuthenticated.set(true); } catch (e) {}
          this.router.navigate(['/dashboard']);
        } else {
          this.success = false;
          this.message = 'Identifiants invalides';
        }
      });
    } else {
      this.message = 'Veuillez remplir tous les champs et sélectionner un rôle';
      this.success = false;
    }
  }

  togglePasswordVisibility() {
    this.showPassword = !this.showPassword;
  }

  // Debug helper: show current location values so we can see fragments / query
  showCurrentUrl() {
    try {
      const href = window.location.href;
      const search = window.location.search;
      const hash = window.location.hash;
  // debug info removed: kept minimal behaviour
      alert('href: ' + href + '\nsearch: ' + search + '\nhash: ' + hash);
    } catch (e) {
  console.error('showCurrentUrl failed', e);
      alert('Impossible de lire l\'URL: ' + e);
    }
  }
}
