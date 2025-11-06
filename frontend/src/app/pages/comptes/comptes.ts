import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ProfileService, Account, SuiviBlacklist } from '../../services/profile.service';
import { Subject, forkJoin } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import * as countries from 'i18n-iso-countries';

// Charger la langue française
import fr from 'i18n-iso-countries/langs/fr.json';

function normalizeStr(str: string): string {
  return str.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase();
}

// Interface locale avec données de suivi
interface AccountWithSuivi extends Account {
  isSurveilled: boolean;
  suiviStatus?: string;
  suiviRaison?: string; // Source du surveillance
  suiviDate?: string;
}

@Component({
  selector: 'app-comptes',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './comptes.html',
  styleUrls: ['./comptes.scss']
})
export class ComptesComponent implements OnInit, OnDestroy {
  comptes: AccountWithSuivi[] = [];
  filteredComptes: AccountWithSuivi[] = [];
  paginatedComptes: AccountWithSuivi[] = [];
  
  suiviMap: Map<string, SuiviBlacklist> = new Map(); // Map numeroCompte -> SuiviBlacklist

  loading = true;
  error = '';
  
  // Pagination
  currentPage = 1;
  itemsPerPage = 5;
  totalPages = 0;

  // Filtres
  searchTerm = '';
  filterPays = '';
  filterType = '';
  filterSurveilles = '';

  // Détails expandables
  expandedAccountId: string | null = null;

  // Tri
  sortBy = 'numeroCompte';
  sortOrder: 'asc' | 'desc' = 'asc';

  private destroy$ = new Subject<void>();

  constructor(private profileService: ProfileService) {}

  ngOnInit() {
    if (typeof window === 'undefined') return;
    // Enregistrer la locale française
    countries.registerLocale(fr);
    this.loadComptes();
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadComptes() {
    this.loading = true;
    this.error = '';
    
    // Charger les comptes et le suivi en parallèle
    forkJoin({
      accounts: this.profileService.getAccounts(),
      suivi: this.profileService.getSuiviBlacklist()
    })
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (result: any) => {
          // Construire la map de suivi
          this.suiviMap.clear();
          if (result.suivi && Array.isArray(result.suivi)) {
            result.suivi.forEach((s: SuiviBlacklist) => {
              if (!this.suiviMap.has(s.numero_compte) || 
                  s.statut_suivi === 'surveiller') {
                this.suiviMap.set(s.numero_compte, s);
              }
            });
          }
          
          // Ajouter les données de suivi aux comptes
          this.comptes = (result.accounts || []).map((account: Account) => ({
            ...account,
            isSurveilled: this.suiviMap.has(account.numeroCompte),
            suiviStatus: this.suiviMap.get(account.numeroCompte)?.statut_suivi,
            suiviRaison: this.suiviMap.get(account.numeroCompte)?.raison,
            suiviDate: this.suiviMap.get(account.numeroCompte)?.date_detection
          }));
          
          this.applyFilters();
          this.loading = false;
        },
        error: (err: any) => {
          this.error = 'Erreur lors du chargement des comptes';
          this.loading = false;
          console.error('Erreur:', err);
          
          // Fallback: charger juste les comptes si suivi échoue
          this.profileService.getAccounts()
            .pipe(takeUntil(this.destroy$))
            .subscribe({
              next: (data: Account[]) => {
                this.comptes = (data || []).map(account => ({
                  ...account,
                  isSurveilled: false
                }));
                this.applyFilters();
              }
            });
        }
      });
  }

  applyFilters() {
    let filtered = [...this.comptes];

    // Filtre de recherche
    if (this.searchTerm) {
      const term = normalizeStr(this.searchTerm);
      filtered = filtered.filter(c =>
        normalizeStr(c.numeroCompte || '').includes(term) ||
        normalizeStr(c.nomClient || '').includes(term) ||
        normalizeStr(c.ville || '').includes(term)
      );
    }

    // Filtre par pays
    if (this.filterPays) {
      const term = normalizeStr(this.filterPays);
      filtered = filtered.filter(c => normalizeStr(c.pays || '') === term);
    }

    // Filtre par type
    if (this.filterType) {
      filtered = filtered.filter(c => (c.typeClient || '').toUpperCase() === this.filterType.toUpperCase());
    }

    // Filtre sur les comptes surveillés
    if (this.filterSurveilles) {
      if (this.filterSurveilles === 'true') {
        filtered = filtered.filter(c => c.isSurveilled === true);
      } else if (this.filterSurveilles === 'false') {
        filtered = filtered.filter(c => c.isSurveilled === false);
      }
    }

    // Tri
    filtered = this.sortComptes(filtered);

    this.filteredComptes = filtered;
    this.currentPage = 1;
    this.updatePagination();
  }

  sortComptes(comptes: AccountWithSuivi[]): AccountWithSuivi[] {
    const multiplier = this.sortOrder === 'asc' ? 1 : -1;
    return comptes.sort((a, b) => {
      let aVal: any = '';
      let bVal: any = '';

      if (this.sortBy === 'numeroCompte') {
        aVal = a.numeroCompte || '';
        bVal = b.numeroCompte || '';
      } else if (this.sortBy === 'nomClient') {
        aVal = a.nomClient || '';
        bVal = b.nomClient || '';
      } else if (this.sortBy === 'dateCreation') {
        aVal = new Date(a.dateCreation).getTime();
        bVal = new Date(b.dateCreation).getTime();
      }

      if (aVal < bVal) return -1 * multiplier;
      if (aVal > bVal) return 1 * multiplier;
      return 0;
    });
  }

  updatePagination() {
    this.totalPages = Math.ceil(this.filteredComptes.length / this.itemsPerPage);
    const startIdx = (this.currentPage - 1) * this.itemsPerPage;
    const endIdx = startIdx + this.itemsPerPage;
    this.paginatedComptes = this.filteredComptes.slice(startIdx, endIdx);
  }

  nextPage() {
    if (this.currentPage < this.totalPages) {
      this.currentPage++;
      this.updatePagination();
    }
  }

  prevPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
      this.updatePagination();
    }
  }

  toggleDetails(account: Account) {
    this.expandedAccountId = this.expandedAccountId === account.id ? null : account.id;
  }

  onFilterChange() {
    this.applyFilters();
  }

  clearFilters() {
    this.searchTerm = '';
    this.filterPays = '';
    this.filterType = '';
    this.filterSurveilles = '';
    this.sortBy = 'numeroCompte';
    this.sortOrder = 'asc';
    this.applyFilters();
  }

  getUniqueCountries(): string[] {
    // Utiliser la liste complète des pays de la bibliothèque i18n-iso-countries
    const countryNames = countries.getNames('fr');
    return Object.values(countryNames).sort() as string[];
  }

  getTypeClientOptions(): { code: string; label: string }[] {
    return [
      { code: 'P', label: 'Particulier' },
      { code: 'M', label: 'Entreprise' },
      { code: 'C', label: 'Commerçant' },
      { code: 'B', label: 'Business' }
    ];
  }

  getTypeClientLabel(code: string): string {
    const options = this.getTypeClientOptions();
    const codeUpper = code?.toUpperCase() || '';
    return options.find(o => o.code === codeUpper)?.label || code;
  }

  // Compter les comptes sous surveillance
  getFollowedCount(): number {
    return this.comptes.filter(c => c.isSurveilled).length;
  }

  dismissError() {
    this.error = '';
  }
}
