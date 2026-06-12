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
        description="Tìm kiếm, lọc theo trạng thái và khoá / mở khoá tài khoản người dùng"
      />
      <div className="p-4 sm:p-6 lg:p-8">
        <UserList />
      </div>
    </>
  );
}
