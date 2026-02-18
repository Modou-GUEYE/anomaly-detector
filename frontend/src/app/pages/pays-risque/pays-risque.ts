import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ProfileService, PaysRisque } from '../../services/profile.service';
import { AuthService } from '../../services/auth.service';
import { ConfigService } from '../../services/config.service';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { MatIconModule } from '@angular/material/icon';


@Component({
  selector: 'app-pays-risque',
  standalone: true,
  imports: [CommonModule, FormsModule, MatIconModule],
  templateUrl: './pays-risque.html',
  styleUrls: ['./pays-risque.scss']
})
export class PaysRisqueComponent implements OnInit, OnDestroy {
  // --- Data ---
  pays: PaysRisque[] = [];
  filteredPays: PaysRisque[] = [];
  paginatedPays: PaysRisque[] = [];

  // --- States ---
  loading = true;
  submitting = false;
  error = '';
  success = '';
  searchTerm = '';

  // --- Modals & UI Controls ---
  showAccessDeniedModal = false;
  adminOnlyMessage = 'Cette action est réservée aux administrateurs';
  showForm = false;
  showEditForm = false;
  showImportForm = false;
  showDeleteConfirmModal = false;
  deletingPays: PaysRisque | null = null;

  // --- Pagination ---
  currentPage = 1;
  itemsPerPage = 10;
  totalPages = 0;

  // --- Forms Data ---
  newPaysName = '';
  newPaysCode = '';
  editingPays: PaysRisque | null = null;
  editedPaysName = '';
  editedPaysCode = '';
  
  // --- Import ---
  selectedFile: File | null = null;
  importMessage = '';
  importError = '';

  // --- Animation Highlight ---
  highlightPaysNom: string | null = null;

  private destroy$ = new Subject<void>();

  constructor(
    private profileService: ProfileService,
    private authService: AuthService,
    private config: ConfigService
  ) {}

  get isAdmin(): boolean {
    return this.authService.isAdmin();
  }

  ngOnInit(): void {
    if (typeof window === 'undefined') return;
    
    // Initialisation du message admin dynamique
    try {
      const adminRole = this.config.getRoleName('admin');
      if (adminRole) this.adminOnlyMessage = `Cette action est réservée au rôle « ${adminRole} »`;
    } catch (e) { }

    this.loadPaysARisque();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  // --- Permissions Check ---
  private checkAdminPermission(): boolean {
    if (!this.isAdmin) {
      this.showAccessDeniedModal = true;
      return false;
    }
    return true;
  }

  closeAccessDeniedModal(): void {
    this.showAccessDeniedModal = false;
  }

  // --- Core Logic ---
  loadPaysARisque(): void {
    this.loading = true;
    this.profileService.getAllPaysARisque()
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (data) => {
          this.pays = data || [];
          this.applyFilters();
          this.loading = false;
        },
        error: () => {
          this.error = 'Erreur lors du chargement des pays';
          this.loading = false;
        }
      });
  }

  applyFilters(): void {
    const term = this.searchTerm.toLowerCase().trim();
    this.filteredPays = this.pays.filter(p => 
      p.nomPays.toLowerCase().includes(term) || 
      (p.code && p.code.toLowerCase().includes(term))
    );
    this.updatePagination();
  }

  onSearch(): void {
    this.currentPage = 1;
    this.applyFilters();
  }

  updatePagination(): void {
    this.totalPages = Math.ceil(this.filteredPays.length / this.itemsPerPage);
    const start = (this.currentPage - 1) * this.itemsPerPage;
    this.paginatedPays = this.filteredPays.slice(start, start + this.itemsPerPage);
  }

  goToPage(page: number): void {
    if (page >= 1 && page <= this.totalPages) {
      this.currentPage = page;
      this.updatePagination();
    }
  }

  // --- Create ---
  toggleForm(): void {
    if (!this.checkAdminPermission()) return;
    this.showForm = !this.showForm;
    this.showEditForm = false;
    this.showImportForm = false;
    if (!this.showForm) this.resetForms();
  }

  addPays(): void {
    if (!this.checkAdminPermission()) return;
    if (!this.newPaysName.trim() || !this.newPaysCode.trim() || !/^[A-Za-z]{2}$/.test(this.newPaysCode.trim())) {
      this.error = 'Format invalide (Nom requis et Code ISO à 2 lettres)';
      return;
    }

    this.submitting = true;
    const nom = this.newPaysName.trim();
    const code = this.newPaysCode.trim().toUpperCase();

    this.profileService.createPaysARisque(nom, code)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => this.handleSuccess(nom, `Pays "${nom}" ajouté avec succès`),
        error: () => this.handleError('Erreur lors de l\'ajout')
      });
  }

  // --- Update ---
  startEdit(pays: PaysRisque): void {
    if (!this.checkAdminPermission()) return;
    this.editingPays = pays;
    this.editedPaysName = pays.nomPays;
    this.editedPaysCode = pays.code || '';
    this.showEditForm = true;
    this.showForm = false;
    this.showImportForm = false;
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  savePaysEdit(): void {
    if (!this.checkAdminPermission() || !this.editingPays) return;
    if (!this.editedPaysName.trim() || !/^[A-Za-z]{2}$/.test(this.editedPaysCode.trim())) {
      this.error = 'Nom requis et Code ISO à 2 lettres';
      return;
    }

    this.submitting = true;
    const nomAncien = this.editingPays.nomPays;
    const nomNouveau = this.editedPaysName.trim();
    const codeNouveau = this.editedPaysCode.trim().toUpperCase();

    this.profileService.updatePaysARisque(nomAncien, nomNouveau, codeNouveau)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => this.handleSuccess(nomNouveau, `Pays mis à jour avec succès`),
        error: () => this.handleError('Erreur lors de la mise à jour')
      });
  }

  // --- Delete ---
  deletePays(pays: PaysRisque): void {
    if (!this.checkAdminPermission()) return;
    this.deletingPays = pays;
    this.showDeleteConfirmModal = true;
  }

  confirmDelete(): void {
    if (!this.deletingPays) return;

    this.submitting = true;
    this.profileService.deletePaysARisque(this.deletingPays.nomPays)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.success = 'Pays supprimé';
          this.loadPaysARisque();
          this.submitting = false;
          this.closeDeleteConfirmModal();
          setTimeout(() => this.success = '', 3000);
        },
        error: () => {
          this.handleError('Erreur lors de la suppression');
          this.closeDeleteConfirmModal();
        }
      });
  }

  closeDeleteConfirmModal(): void {
    this.showDeleteConfirmModal = false;
    this.deletingPays = null;
  }

  // --- Import Logic (Ta logique intégrée) ---
  toggleImportForm(): void {
    if (!this.checkAdminPermission()) return;
    this.showImportForm = !this.showImportForm;
    this.showForm = false;
    this.showEditForm = false;
    if (!this.showImportForm) {
      this.selectedFile = null;
      this.importMessage = '';
      this.importError = '';
    }
  }

  onFileSelected(event: any): void {
    this.selectedFile = event.target.files?.[0] || null;
    this.importError = '';
  }

  importPays(): void {
    if (!this.checkAdminPermission()) return;
    if (!this.selectedFile) {
      this.importError = 'Veuillez sélectionner un fichier';
      return;
    }

    const allowedTypes = [
      'text/csv', 'text/plain', 'application/json',
      'application/vnd.ms-excel',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    ];
    const fileName = this.selectedFile.name.toLowerCase();
    const isExcel = fileName.endsWith('.xlsx') || fileName.endsWith('.xls');

    if (!allowedTypes.includes(this.selectedFile.type) && !isExcel) {
      this.importError = 'Format invalide (CSV, JSON, Excel)';
      return;
    }

    this.submitting = true;
    this.profileService.importPaysARisque(this.selectedFile)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (res) => this.handleImportResult(res),
        error: (err) => {
          this.importError = 'Erreur d\'import : ' + (err.error?.error || 'Serveur injoignable');
          this.submitting = false;
        }
      });
  }

  private handleImportResult(result: any): void {
    if (result.imported > 0) {
      this.success = `${result.imported} pays importés avec succès`;
      this.loadPaysARisque();
      this.showImportForm = false;
      this.submitting = false;
      setTimeout(() => this.success = '', 5000);
    } else {
      this.importError = 'Aucun pays importé';
      this.submitting = false;
    }
  }

  // --- Helpers ---
  private handleSuccess(name: string, msg: string): void {
    this.success = msg;
    this.highlightPaysNom = name;
    this.showForm = this.showEditForm = false;
    this.submitting = false;
    this.loadPaysARisque();
    setTimeout(() => { this.success = ''; this.highlightPaysNom = null; }, 3000);
  }

  private handleError(msg: string): void {
    this.error = msg;
    this.submitting = false;
  }

  private resetForms(): void {
    this.newPaysName = ''; this.newPaysCode = '';
    this.error = ''; this.success = '';
  }

  cancelEdit(): void {
    this.showEditForm = false;
    this.editingPays = null;
  }
}
