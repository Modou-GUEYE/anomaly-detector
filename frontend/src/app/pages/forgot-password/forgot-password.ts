import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { MatIcon } from '@angular/material/icon';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-forgot-password',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule, MatIcon],
  templateUrl: './forgot-password.html',
  styleUrls: ['./forgot-password.scss']
})
export class ForgotPasswordComponent implements OnInit {
  email = '';
  token = '';
  newPassword = '';
  confirmPassword = '';
  loading = false;
  success = false;
  message = '';
  showNewPassword = false;
  showConfirmPassword = false;

  constructor(
    private readonly authService: AuthService,
    private readonly route: ActivatedRoute,
    private readonly router: Router
  ) {}

  ngOnInit(): void {
    this.token = this.route.snapshot.queryParamMap.get('token') || '';
  }

  get isResetMode(): boolean {
    return this.token.length > 0;
  }

  submitForgotPassword(): void {
    this.message = '';
    this.success = false;

    if (!this.email) {
      this.message = 'Veuillez renseigner votre adresse email.';
      return;
    }

    this.loading = true;
    this.authService.forgotPassword(this.email.trim().toLowerCase()).subscribe((ok) => {
      this.loading = false;
      this.success = ok;
      this.message = ok
        ? 'Un email contenant votre lien de réinitialisation vient de vous être envoyé. Pensez à vérifier vos courriers indésirables si besoin.'
        : 'Impossible de traiter votre demande pour le moment. Veuillez réessayer.';
    });
  }

  submitResetPassword(): void {
    this.message = '';
    this.success = false;

    if (!this.newPassword || this.newPassword.length < 6) {
      this.message = 'Le nouveau mot de passe doit contenir au moins 6 caractères.';
      return;
    }

    if (this.newPassword !== this.confirmPassword) {
      this.message = 'La confirmation ne correspond pas au nouveau mot de passe.';
      return;
    }

    this.loading = true;
    this.authService.resetPassword(this.token, this.newPassword).subscribe((ok) => {
      this.loading = false;
      this.success = ok;
      this.message = ok
        ? 'Votre mot de passe a été réinitialisé avec succès. Redirection en cours...'
        : 'Ce lien de réinitialisation est invalide ou a expiré.';

      if (ok) {
        setTimeout(() => this.router.navigate(['/login']), 1200);
      }
    });
  }
}
