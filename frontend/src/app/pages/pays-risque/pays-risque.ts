import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ProfileService, PaysRisque } from '../../services/profile.service';
import { AuthService } from '../../services/auth.service';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

@Component({
  selector: 'app-pays-risque',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './pays-risque.html',
  styleUrls: ['./pays-risque.scss']
})
export class PaysRisqueComponent implements OnInit, OnDestroy {
  pays: PaysRisque[] = [];
  filteredPays: PaysRisque[] = [];
  paginatedPays: PaysRisque[] = [];

  loading = true;
  error = '';
  success = '';

  // Modal d'accès refusé (comme dans control-rules)
  showAccessDeniedModal = false;
  adminOnlyMessage = 'Cette action est réservée aux administrateurs';

  // Pagination
  currentPage = 1;
  itemsPerPage = 10;
  totalPages = 0;

  // Filtres
  searchTerm = '';

  // Formulaire d'ajout
  newPaysName = '';
  newPaysCode = ''; // Code ISO alpha-2
  showForm = false;
  submitting = false;

  // Formulaire d'édition
  editingPays: PaysRisque | null = null;
  editedPaysName = '';
  editedPaysCode = ''; // Code ISO alpha-2
  showEditForm = false;

  // Pays récemment ajouté / modifié pour mise en surbrillance
  highlightPaysNom: string | null = null;

  // Import
  showImportForm = false;
  selectedFile: File | null = null;
  importMessage = '';
  importError = '';

  private destroy$ = new Subject<void>();

  constructor(
    private profileService: ProfileService,
    private authService: AuthService
  ) {}

  /**
   * Vérifie si l'utilisateur connecté est un administrateur
   */
  get isAdmin(): boolean {
    return this.authService.isAdmin();
  }

  /**
   * Affiche le modal d'accès refusé
   */
  showAccessDenied(): void {
    this.showAccessDeniedModal = true;
  }

  /**
   * Ferme le modal d'accès refusé
   */
  closeAccessDeniedModal(): void {
    this.showAccessDeniedModal = false;
  }

  ngOnInit() {
    if (typeof window === 'undefined') return;
    this.loadPaysARisque();
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  /**
   * Vérifie si l'utilisateur a les permissions admin
   * Affiche un modal si ce n'est pas le cas
   */
  private checkAdminPermission(): boolean {
    if (!this.authService.isAdmin()) {
      this.showAccessDenied();
      return false;
    }
    return true;
  }

  loadPaysARisque() {
    this.loading = true;
    this.error = '';
    this.success = '';

    this.profileService.getAllPaysARisque()
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (data: PaysRisque[]) => {
          this.pays = data || [];
          this.applyFilters();
          this.loading = false;
        },
        error: () => {
          this.error = 'Erreur lors du chargement des pays à risque';
          this.loading = false;
        }
      });
  }

  applyFilters() {
    let filtered = [...this.pays];

    if (this.searchTerm) {
      const term = this.searchTerm.toLowerCase();
      filtered = filtered.filter(p =>
        p.nomPays.toLowerCase().includes(term) ||
        (p.code && p.code.toLowerCase().includes(term))
      );
    }

    this.filteredPays = filtered;
    this.updatePagination();
  }

  updatePagination() {
    this.totalPages = Math.ceil(this.filteredPays.length / this.itemsPerPage);
    const start = (this.currentPage - 1) * this.itemsPerPage;
    const end = start + this.itemsPerPage;
    this.paginatedPays = this.filteredPays.slice(start, end);
  }

  onSearch() {
    this.currentPage = 1;
    this.applyFilters();
  }

  goToPage(page: number) {
    if (page >= 1 && page <= this.totalPages) {
      this.currentPage = page;
      this.updatePagination();
    }
  }

  toggleForm() {
    // Vérifier les permissions avant d'afficher le formulaire
    if (!this.checkAdminPermission()) {
      return;
    }
    
    this.showForm = !this.showForm;
    if (!this.showForm) {
      this.newPaysName = '';
      this.newPaysCode = '';
    }
  }

  addPays() {
    // Vérifier les permissions
    if (!this.checkAdminPermission()) {
      return;
    }
    
    if (!this.newPaysName.trim()) {
      this.error = 'Veuillez entrer un nom de pays';
      return;
    }

    // Validation du code (OBLIGATOIRE)
    if (!this.newPaysCode || !/^[A-Za-z]{2}$/.test(this.newPaysCode.trim())) {
      this.error = 'Le code pays doit être ISO alpha-2 (2 lettres)';
      return;
    }

    this.submitting = true;
    this.error = '';

    const nomAjoute = this.newPaysName.trim();
    const codeAjoute = this.newPaysCode.trim().toUpperCase();
    this.profileService.createPaysARisque(nomAjoute, codeAjoute)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.success = `Pays "${nomAjoute}" (${codeAjoute}) ajouté avec succès`;
          this.newPaysName = '';
          this.newPaysCode = '';
          this.showForm = false;
          this.submitting = false;
          this.highlightPaysNom = nomAjoute;
          this.loadPaysARisque();
          setTimeout(() => { this.success = ''; this.highlightPaysNom = null; }, 3000);
        },
        error: () => {
          this.error = 'Erreur lors de l\'ajout du pays';
          this.submitting = false;
        }
      });
  }

  deletePays(pays: PaysRisque) {
    // Vérifier les permissions
    if (!this.checkAdminPermission()) {
      return;
    }
    if (this.submitting) return; // Guard double clic
    
    if (!confirm(`Êtes-vous sûr de vouloir supprimer "${pays.nomPays}" ?`)) {
      return;
    }

    this.submitting = true;
    this.profileService.deletePaysARisque(pays.nomPays)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.success = `Pays "${pays.nomPays}" supprimé avec succès`;
          this.loadPaysARisque();
          this.submitting = false;
          setTimeout(() => this.success = '', 3000);
        },
        error: () => {
          this.error = 'Erreur lors de la suppression du pays';
          this.submitting = false;
        }
      });
  }

  startEdit(pays: PaysRisque) {
    // Vérifier les permissions
    if (!this.checkAdminPermission()) {
      return;
    }
    
    this.editingPays = pays;
    this.editedPaysName = pays.nomPays;
    this.editedPaysCode = pays.code || '';
    this.showEditForm = true;
  }

  cancelEdit() {
    this.editingPays = null;
    this.editedPaysName = '';
    this.editedPaysCode = '';
    this.showEditForm = false;
  }

  savePaysEdit() {
    // Vérifier les permissions
    if (!this.checkAdminPermission()) {
      return;
    }
    
    if (!this.editedPaysName.trim()) {
      this.error = 'Veuillez entrer un nom de pays';
      return;
    }

    if (!this.editingPays) {
      return;
    }

    // Validation du code (OBLIGATOIRE)
    if (!this.editedPaysCode || !/^[A-Za-z]{2}$/.test(this.editedPaysCode.trim())) {
      this.error = 'Le code pays doit être ISO alpha-2 (2 lettres)';
      return;
    }

    this.submitting = true;
    this.error = '';

    const nomAncien = this.editingPays.nomPays;
    const nomNouveau = this.editedPaysName.trim();
    const codeNouveau = this.editedPaysCode.trim().toUpperCase();
    this.profileService.updatePaysARisque(nomAncien, nomNouveau, codeNouveau)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.success = `Pays "${nomAncien}" mis à jour en "${nomNouveau}" (${codeNouveau})`;
          this.editingPays = null;
          this.editedPaysName = '';
          this.editedPaysCode = '';
          this.showEditForm = false;
          this.submitting = false;
          this.highlightPaysNom = nomNouveau;
          this.loadPaysARisque();
          setTimeout(() => { this.success = ''; this.highlightPaysNom = null; }, 3000);
        },
        error: () => {
          this.error = 'Erreur lors de la mise à jour du pays';
          this.submitting = false;
        }
      });
  }

  toggleImportForm() {
    // Vérifier les permissions avant d'afficher le formulaire d'import
    if (!this.checkAdminPermission()) {
      return;
    }
    
    this.showImportForm = !this.showImportForm;
    if (!this.showImportForm) {
      this.selectedFile = null;
      this.importMessage = '';
      this.importError = '';
    }
  }

  onFileSelected(event: any) {
    this.selectedFile = event.target.files?.[0] || null;
    if (this.selectedFile) {
      this.importError = '';
    }
  }

  importPays() {
    // Vérifier les permissions
    if (!this.checkAdminPermission()) {
      return;
    }
    
    if (!this.selectedFile) {
      this.importError = 'Veuillez sélectionner un fichier';
      return;
    }
    if (this.submitting) return; // Guard double clic import

    const allowedTypes = [
      'text/csv', 
      'text/plain', 
      'application/json', 
      'application/vnd.ms-excel',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    ];
    
    const fileName = this.selectedFile.name.toLowerCase();
    const isExcel = fileName.endsWith('.xlsx') || fileName.endsWith('.xls');
    const isValidType = allowedTypes.includes(this.selectedFile.type) || isExcel;
    
    if (!isValidType) {
      this.importError = 'Format de fichier non autorisé. Utilisez CSV, JSON ou Excel (.xlsx, .xls).';
      return;
    }

    this.submitting = true;
    this.importError = '';
    this.importMessage = '';

    // Envoyer tous les formats au backend (CSV, JSON, Excel)
    this.profileService.importPaysARisque(this.selectedFile)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (result: any) => {
          this.handleImportResult(result);
        },
        error: (err: any) => {
          this.importError = 'Erreur lors de l\'import: ' + (err.error?.error || err.message);
          this.submitting = false;
        }
      });
  }

  private handleImportResult(result: any) {
    if (result.imported > 0) {
      this.importMessage = `${result.imported}/${result.total} pays importés avec succès`;
      if (result.errors && result.errors.length > 0) {
        this.importMessage += `. ${result.errors.length} erreur(s).`;
      }
      this.loadPaysARisque();
      this.selectedFile = null;
      this.showImportForm = false;
      this.submitting = false;
      setTimeout(() => this.importMessage = '', 5000);
    } else {
      this.importError = 'Aucun pays n\'a pu être importé';
      this.submitting = false;
    }
  }

  dismissError() {
    this.error = '';
  }

  dismissSuccess() {
    this.success = '';
  }
}
