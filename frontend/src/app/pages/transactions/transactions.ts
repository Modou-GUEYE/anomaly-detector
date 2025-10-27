import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-transactions',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './transactions.html',
  styleUrls: ['./transactions.scss']
})
export class TransactionsComponent implements OnInit {
  transactions: any[] = [];

  ngOnInit() {
    // N'exécuter côté client uniquement
    if (typeof window === 'undefined') {
      return;
    }
    // TODO: Charger les transactions depuis le backend
    this.loadTransactions();
  }

  loadTransactions() {
    // Vous implémenterez le service ici
  }
}
