import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { BlacklistService, EntiteBlacklist } from '../../services/blacklist.service';

interface Filter {
  search: string;
  typeEntite: string;
}

@Component({
  selector: 'app-blacklist',
  standalone: true,
  imports: [CommonModule, FormsModule],
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
  pageSize = 5;
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

  saveEntity() {
    if (!this.formData.nom || !this.formData.numeroCompte || !this.formData.typeEntite) {
      alert('Veuillez remplir tous les champs obligatoires');
      return;
    }

    if (this.isEditMode && this.editingEntity?.id) {
      this.blacklistService.updateEntity(this.editingEntity.id, this.formData).subscribe({
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

  deleteEntity(id?: string) {
    if (!id) return;
    if (!confirm('Êtes-vous sûr de vouloir supprimer cette entité ?')) return;

    this.blacklistService.deleteEntity(id).subscribe({
      next: () => {
        this.loadEntities();
      },
      error: (err) => {
        alert('Erreur lors de la suppression');
      }
    });
  }

  toggleExpand(entity: any) {
    entity.expanded = !entity.expanded;
  }

  padNumber(num: number): string {
    return num.toString().padStart(2, '0');
  }
}
