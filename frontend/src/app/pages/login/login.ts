
import { Component } from '@angular/core';
import { AuthService } from '../../services/auth.service';
import { AuthStateService } from '../../services/auth-state.service';
import { Router, RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { ConfigService } from '../../services/config.service';
import { MatIcon } from '@angular/material/icon';


@Component({
  selector: 'app-login',
  standalone: true,
  templateUrl: './login.html',
  styleUrls: ['./login.scss'],
  imports: [CommonModule, FormsModule, RouterModule, MatIcon],
})
export class LoginComponent {
  email: string = '';
  password: string = '';
  message: string = '';
  success: boolean = false;
  isLoadingMessage: boolean = false;
  showPassword: boolean = false;

  constructor(private authService: AuthService, private authState: AuthStateService, private router: Router, private config: ConfigService) {}

  onSubmit() {
    // Handler du formulaire de connexion
    this.message = '';
    this.success = false;
    this.isLoadingMessage = false;
    if (this.email && this.password) {
      this.isLoadingMessage = true;
      this.message = 'Connexion...';
      // Convertir l'email en minuscules pour être insensible à la casse
      const emailLowerCase = this.email.toLowerCase();
      // Le rôle est déterminé automatiquement côté serveur
      this.authService.login(emailLowerCase, this.password).subscribe((ok) => {
        if (ok) {
          this.isLoadingMessage = false;
          this.success = true;
          try { this.authState.isAuthenticated.set(true); } catch (e) {}
          
          // Vérifier si l'utilisateur doit changer son mot de passe
          if (this.authService.mustChangePassword()) {
            this.router.navigate(['/first-password-change']);
          } else {
            this.router.navigate(['/dashboard']);
          }
        } else {
          this.isLoadingMessage = false;
          this.success = false;
          this.message = 'Identifiants invalides';
        }
      });
    } else {
      this.message = 'Veuillez remplir tous les champs';
      this.success = false;
      this.isLoadingMessage = false;
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
