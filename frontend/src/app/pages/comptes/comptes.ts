import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-comptes',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './comptes.html',
  styleUrls: ['./comptes.scss']
})
export class ComptesComponent implements OnInit {
  comptes: any[] = [];

  ngOnInit() {
    // N'exécuter côté client uniquement
    if (typeof window === 'undefined') {
      return;
    }
    // TODO: Charger les comptes depuis le backend
    this.loadComptes();
  }

  loadComptes() {
    // Vous implémenterez le service ici
  }
}
