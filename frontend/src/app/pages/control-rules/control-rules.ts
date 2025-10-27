import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-control-rules',
  standalone: true,
  templateUrl: './control-rules.html',
  styleUrls: ['./control-rules.scss'],
  imports: [CommonModule, FormsModule]
})
export class ControlRulesComponent implements OnInit {
  loading = true;
  error = '';
  
  rules: any[] = [
    {
      id: 1,
      name: 'Montant Maximum',
      description: 'Détecter les transactions dépassant 10 000 000 FCFA',
      type: 'Montant',
      status: 'Actif',
      threshold: 10000000
    },
    {
      id: 2,
      name: 'Fréquence Anormale',
      description: 'Plus de 5 transactions en 1 heure',
      type: 'Fréquence',
      status: 'Actif',
      threshold: 5
    },
    {
      id: 3,
      name: 'Pays à Risque',
      description: 'Transactions vers certains pays',
      type: 'Géographie',
      status: 'Actif',
      threshold: 0
    }
  ];

  constructor() {}

  ngOnInit(): void {
    // Simuler le chargement
    setTimeout(() => {
      this.loading = false;
    }, 500);
  }

  editRule(rule: any): void {
  }

  deleteRule(id: number): void {
    this.rules = this.rules.filter(r => r.id !== id);
  }

  addRule(): void {
  }
}
