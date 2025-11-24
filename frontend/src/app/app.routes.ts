import { Routes } from '@angular/router';
import { LoginComponent } from './pages/login/login';
import { SigninComponent } from './pages/signin/signin';
import { DashboardComponent } from './pages/dashboard/dashboard';
import { AnomaliesComponent } from './pages/anomalies/anomalies';
import { ComptesComponent } from './pages/comptes/comptes';
import { TransactionsComponent } from './pages/transactions/transactions';
import { BlacklistComponent } from './pages/blacklist/blacklist';
import { ControlRulesComponent } from './pages/control-rules/control-rules';
import { ReportsComponent } from './pages/reports/reports';
import { PaysRisqueComponent } from './pages/pays-risque/pays-risque';
import { TrackingComponent } from './pages/tracking/tracking';
import { CreateUserComponent } from './pages/create-user/create-user';
import { ParametreComponent } from './pages/parametre/parametre';
import { AuthGuard } from './guards/auth.guard';
import { RoleGuard } from './guards/role.guard';
import { LayoutComponent } from './layouts/layout/layout';
export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  { path: 'signin', component: SigninComponent },
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
      { path: 'pays-risque', component: PaysRisqueComponent },
      { path: 'reports', component: ReportsComponent },
  { path: 'tracking', component: TrackingComponent, canActivate: [AuthGuard, RoleGuard], data: { forbiddenRoles: ['analyst', 'analyste'] } },
  { path: 'create-user', component: CreateUserComponent, canActivate: [AuthGuard, RoleGuard], data: { allowedRoles: ['admin'] } },
  { path: 'parametre', component: ParametreComponent },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' }
    ]
  },
  { path: '', redirectTo: 'login', pathMatch: 'full' }
];

