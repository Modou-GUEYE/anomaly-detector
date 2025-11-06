import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ProfileService } from '../../services/profile.service';
import { AuthService } from '../../services/auth.service';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import * as XLSX from 'xlsx';

@Component({
  selector: 'app-pays-risque',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './pays-risque.html',
  styleUrls: ['./pays-risque.scss']
})
export class PaysRisqueComponent implements OnInit, OnDestroy {
  pays: string[] = [];
  filteredPays: string[] = [];
  paginatedPays: string[] = [];

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
  showForm = false;
  submitting = false;

  // Formulaire d'édition
  editingPays: string | null = null;
  editedPaysName = '';
  showEditForm = false;

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
        next: (data: string[]) => {
          this.pays = data || [];
          this.applyFilters();
          this.loading = false;
        },
        error: (err: any) => {
          this.error = 'Erreur lors du chargement des pays à risque';
          console.error('Erreur:', err);
          this.loading = false;
        }
      });
  }

  applyFilters() {
    let filtered = [...this.pays];

    if (this.searchTerm) {
      const term = this.searchTerm.toLowerCase();
      filtered = filtered.filter(p =>
        p.toLowerCase().includes(term)
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

    this.submitting = true;
    this.error = '';

    this.profileService.createPaysARisque(this.newPaysName.trim())
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.success = `Pays "${this.newPaysName}" ajouté avec succès`;
          this.newPaysName = '';
          this.showForm = false;
          this.submitting = false;
          this.loadPaysARisque();
          setTimeout(() => this.success = '', 3000);
        },
        error: (err: any) => {
          this.error = 'Erreur lors de l\'ajout du pays';
          this.submitting = false;
          console.error('Erreur:', err);
        }
      });
  }

  deletePays(nomPays: string) {
    // Vérifier les permissions
    if (!this.checkAdminPermission()) {
      return;
    }
    
    if (!confirm(`Êtes-vous sûr de vouloir supprimer "${nomPays}" ?`)) {
      return;
    }

    this.profileService.deletePaysARisque(nomPays)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.success = `Pays "${nomPays}" supprimé avec succès`;
          this.loadPaysARisque();
          setTimeout(() => this.success = '', 3000);
        },
        error: (err: any) => {
          this.error = 'Erreur lors de la suppression du pays';
          console.error('Erreur:', err);
        }
      });
  }

  startEdit(nomPays: string) {
    // Vérifier les permissions
    if (!this.checkAdminPermission()) {
      return;
    }
    
    this.editingPays = nomPays;
    this.editedPaysName = nomPays;
    this.showEditForm = true;
  }

  cancelEdit() {
    this.editingPays = null;
    this.editedPaysName = '';
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

    this.submitting = true;
    this.error = '';

    this.profileService.updatePaysARisque(this.editingPays, this.editedPaysName.trim())
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          this.success = `Pays "${this.editingPays}" renommé en "${this.editedPaysName}" avec succès`;
          this.editingPays = null;
          this.editedPaysName = '';
          this.showEditForm = false;
          this.submitting = false;
          this.loadPaysARisque();
          setTimeout(() => this.success = '', 3000);
        },
        error: (err: any) => {
          this.error = 'Erreur lors de la mise à jour du pays';
          this.submitting = false;
          console.error('Erreur:', err);
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

    // Si c'est un fichier Excel, on le traite localement
    if (isExcel) {
      this.processExcelFile();
    } else {
      // Sinon, on utilise l'API existante
      this.profileService.importPaysARisque(this.selectedFile)
        .pipe(takeUntil(this.destroy$))
        .subscribe({
          next: (result: any) => {
            this.handleImportResult(result);
          },
          error: (err: any) => {
            this.importError = 'Erreur lors de l\'import: ' + (err.error?.error || err.message);
            this.submitting = false;
            console.error('Erreur:', err);
          }
        });
    }
  }

  private processExcelFile() {
    const reader = new FileReader();
    reader.onload = (e: any) => {
      try {
        const data = new Uint8Array(e.target.result);
        const workbook = XLSX.read(data, { type: 'array' });
        
        // Prendre la première feuille
        const firstSheetName = workbook.SheetNames[0];
        const worksheet = workbook.Sheets[firstSheetName];
        
        // Convertir en JSON
        const jsonData = XLSX.utils.sheet_to_json(worksheet, { header: 1 });
        
        // Extraire les noms de pays
        const paysToImport = this.extractPaysFromExcelData(jsonData);
        
        if (paysToImport.length === 0) {
          this.importError = 'Aucun pays trouvé dans le fichier Excel. Assurez-vous que la première colonne contient les noms de pays.';
          this.submitting = false;
          return;
        }

        // Importer les pays un par un
        this.importPaysArray(paysToImport);
        
      } catch (error) {
        this.importError = 'Erreur lors de la lecture du fichier Excel: ' + error;
        this.submitting = false;
        console.error('Erreur Excel:', error);
      }
    };
    
    reader.readAsArrayBuffer(this.selectedFile!);
  }

  private extractPaysFromExcelData(data: any[]): string[] {
    const pays: string[] = [];
    
    // Ignorer la première ligne si elle contient des en-têtes
    const startRow = data.length > 0 && typeof data[0][0] === 'string' && 
                     (data[0][0].toLowerCase().includes('pays') || 
                      data[0][0].toLowerCase().includes('country') ||
                      data[0][0].toLowerCase().includes('nom')) ? 1 : 0;
    
    for (let i = startRow; i < data.length; i++) {
      const row = data[i];
      if (row && row[0] && typeof row[0] === 'string') {
        const paysName = row[0].toString().trim();
        if (paysName && !pays.includes(paysName)) {
          pays.push(paysName);
        }
      }
    }
    
    return pays;
  }

  private importPaysArray(paysArray: string[]) {
    let imported = 0;
    let errors = 0;
    let processed = 0;
    
    const importNext = () => {
      if (processed >= paysArray.length) {
        // Tous les pays ont été traités
        const result = {
          imported: imported,
          total: paysArray.length,
          errors: errors > 0 ? [`${errors} pays n'ont pas pu être importés`] : []
        };
        this.handleImportResult(result);
        return;
      }
      
      const pays = paysArray[processed];
      processed++;
      
      this.profileService.createPaysARisque(pays)
        .pipe(takeUntil(this.destroy$))
        .subscribe({
          next: () => {
            imported++;
            importNext();
          },
          error: (err: any) => {
            errors++;
            console.error(`Erreur pour ${pays}:`, err);
            importNext();
          }
        });
    };
    
    importNext();
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
