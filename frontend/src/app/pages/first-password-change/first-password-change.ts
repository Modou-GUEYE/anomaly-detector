import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { AuthService } from '../../services/auth.service';
import { HttpErrorResponse } from '@angular/common/http';

@Component({
  selector: 'app-first-password-change',
  standalone: true,
  imports: [CommonModule, FormsModule, MatIconModule],
  templateUrl: './first-password-change.html',
  styleUrls: ['./first-password-change.scss']
})
export class FirstPasswordChangeComponent {
  oldPassword = '';
  newPassword = '';
  confirmPassword = '';
  showOldPassword = false;
  showNewPassword = false;
  showConfirmPassword = false;
  loading = false;
  errorMessage = '';

  constructor(
    private authService: AuthService,
    private router: Router
  ) {}

  toggleOldPassword() {
    this.showOldPassword = !this.showOldPassword;
  }

  toggleNewPassword() {
    this.showNewPassword = !this.showNewPassword;
  }

  toggleConfirmPassword() {
    this.showConfirmPassword = !this.showConfirmPassword;
  }

  onSubmit() {
    this.errorMessage = '';

    // Validations
    if (!this.oldPassword) {
      this.errorMessage = 'Veuillez saisir votre mot de passe temporaire.';
      return;
    }

    if (!this.newPassword || this.newPassword.length < 6) {
      this.errorMessage = 'Le nouveau mot de passe doit contenir au moins 6 caractères.';
      return;
    }

    if (this.newPassword !== this.confirmPassword) {
      this.errorMessage = 'La confirmation ne correspond pas au nouveau mot de passe.';
      return;
    }

    if (this.oldPassword === this.newPassword) {
      this.errorMessage = 'Le nouveau mot de passe doit être différent de l\'ancien.';
      return;
    }

    this.loading = true;

    this.authService.changeFirstPassword(this.oldPassword, this.newPassword).subscribe({
      next: (success) => {
        this.loading = false;
        if (success) {
          // Rediriger vers le dashboard
          this.router.navigate(['/dashboard']);
        } else {
          this.errorMessage = 'Une erreur est survenue lors du changement de mot de passe.';
        }
      },
      error: (err: HttpErrorResponse) => {
        this.loading = false;
        this.errorMessage = this.getErrorMessage(err);
      }
    });
  }

  private getErrorMessage(error: HttpErrorResponse): string {
    if (error.status === 401) {
      return 'Le mot de passe temporaire est incorrect.';
    }
    if (error.status === 400) {
      return 'Les données fournies sont invalides.';
    }
    if (error.status === 0) {
      return 'Le serveur est indisponible. Veuillez réessayer plus tard.';
    }
    return error.error?.error || 'Une erreur est survenue lors du changement de mot de passe.';
  }
}
