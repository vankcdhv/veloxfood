import { redirect } from 'next/navigation';

// /account has no landing of its own — send users to their orders.
export default function AccountIndexPage() {
  redirect('/account/orders');
}
