import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { BlacklistService, EntiteBlacklist } from '../../services/blacklist.service';
import { MatIconModule } from '@angular/material/icon';

interface Filter {
  search: string;
  typeEntite: string;
}

@Component({
  selector: 'app-blacklist',
  standalone: true,
  imports: [CommonModule, FormsModule,MatIconModule],
  templateUrl: './blacklist.html',
  styleUrls: ['./blacklist.scss']
})
export class BlacklistComponent implements OnInit {
  entities: EntiteBlacklist[] = [];
  filteredEntities: EntiteBlacklist[] = [];
  loading = true;
  error = '';

  // Pagination
  currentPage = 1;
  pageSize = 10;
  totalPages = 1;

  // Filtres
  filters: Filter = {
    search: '',
    typeEntite: ''
  };

  // Modal
  showModal = false;
  isEditMode = false;
  editingEntity: EntiteBlacklist | null = null;
  showDeleteModal = false;
  deletingEntity: EntiteBlacklist | null = null;
  showDetailsModal = false;
  detailsEntity: EntiteBlacklist | null = null;
  formData: EntiteBlacklist = {
    nom: '',
    numeroCompte: '',
    typeEntite: '',
    informationsSupplementaires: ''
  };

  // Options
  typeOptions = ['Blacklist', 'PEP', 'Surveillés', 'Interdit'];

  constructor(private blacklistService: BlacklistService) {}

  ngOnInit() {
    if (typeof window === 'undefined') {
      return;
    }
    this.loadEntities();
  }

  loadEntities() {
    this.loading = true;
    this.blacklistService.getEntities().subscribe({
      next: (data) => {
        this.entities = data || [];
        // Trier par date décroissante (plus récentes en premier)
        this.entities.sort((a, b) => {
          const dateA = a.dateAjout ? new Date(a.dateAjout).getTime() : 0;
          const dateB = b.dateAjout ? new Date(b.dateAjout).getTime() : 0;
          return dateB - dateA;
        });
        this.applyFilters();
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Impossible de charger les entités';
        this.loading = false;
      }
    });
  }

  applyFilters() {
    let result = [...this.entities];

    // Recherche
    if (this.filters.search) {
      const search = this.filters.search.toLowerCase();
      result = result.filter(e =>
        e.nom.toLowerCase().includes(search) ||
        e.numeroCompte.toLowerCase().includes(search)
      );
    }

    // Filtre type
    if (this.filters.typeEntite) {
      result = result.filter(e => e.typeEntite === this.filters.typeEntite);
    }

    this.totalPages = Math.ceil(result.length / this.pageSize);
    this.currentPage = 1;
    this.filteredEntities = result;
  }

  get paginatedEntities() {
    const start = (this.currentPage - 1) * this.pageSize;
    const end = start + this.pageSize;
    return this.filteredEntities.slice(start, end);
  }

  nextPage() {
    if (this.currentPage < this.totalPages) {
      this.currentPage++;
    }
  }

  prevPage() {
    if (this.currentPage > 1) {
      this.currentPage--;
    }
  }

  openAddModal() {
    this.isEditMode = false;
    this.formData = {
      nom: '',
      numeroCompte: '',
      typeEntite: '',
      informationsSupplementaires: ''
    };
    this.showModal = true;
  }

  openEditModal(entity: EntiteBlacklist) {
    this.isEditMode = true;
    this.editingEntity = entity;
    this.formData = { ...entity };
    this.showModal = true;
  }

  closeModal() {
    this.showModal = false;
    this.editingEntity = null;
  }

  openDeleteModal(entity: EntiteBlacklist) {
    this.deletingEntity = entity;
    this.showDeleteModal = true;
  }

  closeDeleteModal() {
    this.showDeleteModal = false;
    this.deletingEntity = null;
  }

  openDetailsModal(entity: EntiteBlacklist) {
    this.detailsEntity = entity;
    this.showDetailsModal = true;
  }

  closeDetailsModal() {
    this.showDetailsModal = false;
    this.detailsEntity = null;
  }

  private getEntityId(entity?: EntiteBlacklist | null): string | null {
    const anyEntity = entity as any;
    return anyEntity?.id || anyEntity?._id || null;
  }

  saveEntity() {
    if (!this.formData.nom || !this.formData.numeroCompte || !this.formData.typeEntite) {
      alert('Veuillez remplir tous les champs obligatoires');
      return;
    }

    if (this.isEditMode) {
      const id = this.getEntityId(this.editingEntity);
      if (!id) {
        alert('Identifiant introuvable pour cette entite');
        return;
      }
      this.blacklistService.updateEntity(id, this.formData).subscribe({
        next: () => {
          this.loadEntities();
          this.closeModal();
        },
        error: (err) => {
          alert('Erreur lors de la mise à jour');
        }
      });
    } else {
      this.blacklistService.createEntity(this.formData).subscribe({
        next: () => {
          this.loadEntities();
          this.closeModal();
        },
        error: (err) => {
          alert('Erreur lors de la création');
        }
      });
    }
  }

  confirmDelete() {
    const id = this.getEntityId(this.deletingEntity);
    if (!id) return;

    this.blacklistService.deleteEntity(id).subscribe({
      next: () => {
        this.loadEntities();
        this.closeDeleteModal();
      },
      error: () => {
        alert('Erreur lors de la suppression');
      }
    });
  }


  padNumber(num: number): string {
    return num.toString().padStart(2, '0');
  }

  getEntityIcon(type: string): string {
    switch (type.toLowerCase()) {
      case 'blacklist': return 'block';
      case 'pep': return 'person_search';
      case 'interdit': return 'gavel';
      default: return 'person';
    }
  }
  
}
