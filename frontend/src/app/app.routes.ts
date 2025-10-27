import { Routes } from '@angular/router';
import { LoginComponent } from './pages/login/login';
import { DashboardComponent } from './pages/dashboard/dashboard';
import { AnomaliesComponent } from './pages/anomalies/anomalies';
import { ComptesComponent } from './pages/comptes/comptes';
import { TransactionsComponent } from './pages/transactions/transactions';
import { BlacklistComponent } from './pages/blacklist/blacklist';
import { ControlRulesComponent } from './pages/control-rules/control-rules';
import { ReportsComponent } from './pages/reports/reports';
import { AuthGuard } from './guards/auth.guard';
import { LayoutComponent } from './layouts/layout/layout';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  {
    path: '',
    component: LayoutComponent,
    canActivate: [AuthGuard],
    children: [
      { path: 'dashboard', component: DashboardComponent },
      { path: 'transactions', component: TransactionsComponent },
      { path: 'comptes', component: ComptesComponent },
      { path: 'anomalies', component: AnomaliesComponent },
      { path: 'control-rules', component: ControlRulesComponent },
      { path: 'blacklist', component: BlacklistComponent },
      { path: 'reports', component: ReportsComponent },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' }
    ]
  },
  { path: '', redirectTo: 'login', pathMatch: 'full' }
];

