
import { Component } from '@angular/core';
import { AuthService } from '../../services/auth.service';
import { Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-login',
  templateUrl: './login.html',
  styleUrls: ['./login.scss'],
  imports: [CommonModule, FormsModule],
  providers: [AuthService]
})
export class LoginComponent {
  email: string = '';
  password: string = '';
  message: string = '';
  success: boolean = false;

  constructor(private authService: AuthService, private router: Router) {}

  onSubmit() {
    this.message = '';
    this.success = false;
    if (this.email && this.password) {
      this.authService.login(this.email, this.password).subscribe({
        next: (response: any) => {
          this.success = true;
          this.message = 'Connexion réussie !';
          if (response && response.token) {
            localStorage.setItem('token', response.token);
            setTimeout(() => {
              this.router.navigate(['/dashboard']);
            }, 1000);
          }
        },
        error: (err: any) => {
          this.success = false;
          this.message = 'Échec de la connexion';
        }
      });
    } else {
      this.message = 'Veuillez remplir tous les champs';
      this.success = false;
    }
  }
}
