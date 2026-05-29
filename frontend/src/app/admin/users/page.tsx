import { UserList } from '@/features/users/components/user-list';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';

export const metadata = {
  title: 'Người dùng',
};

export default function AdminUsersPage() {
  return (
    <>
      <AdminTopbar
        title="Quản lý người dùng"
        description="Demo gọi GET /api/v1/users backend Go qua TanStack Query + Axios"
      />
      <div className="p-4 sm:p-6 lg:p-8">
        <UserList />
      </div>
    </>
  );
}
