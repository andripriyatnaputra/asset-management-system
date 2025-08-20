// File: src/components/Header.tsx
import { Link, useNavigate } from 'react-router-dom';
import { Menu, Package, CircleUser, PanelLeftClose } from 'lucide-react';
import { useState } from 'react';

import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import ChangePasswordModal from '../components/ChangePasswordModal';
import { getNavItems } from '../config/nav';
import { NavLink } from 'react-router-dom';

interface HeaderProps {
  setIsCollapsed: React.Dispatch<React.SetStateAction<boolean>>;
}

export default function Header({ setIsCollapsed }: HeaderProps) {
  const navigate = useNavigate();
  const [isPasswordModalOpen, setIsPasswordModalOpen] = useState(false);
  
  const navItems = getNavItems();
  const handleLogout = () => {
    localStorage.removeItem('authToken');
    navigate('/login');
  };

  return (
    <>
    <header className="flex h-14 items-center gap-4 border-b bg-muted/40 px-4 lg:h-[60px] lg:px-6">
      <Sheet>
        <SheetTrigger asChild>
          <Button variant="outline" size="icon" className="shrink-0 md:hidden">
            <Menu className="h-6 w-6" />
            <span className="sr-only">Toggle navigation menu</span>
          </Button>
        </SheetTrigger>
        <SheetContent side="left" className="flex flex-col">
          <nav className="grid gap-2 text-lg font-medium">
            <Link to="/" className="flex items-center gap-2 text-lg font-semibold mb-5">
              <Package className="h-3 w-3" />
              <span>IT-AMS</span>
            </Link>
            {navItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  `mx-[-0.25rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground
                  ${isActive && 'bg-muted text-foreground'} items center`
                }
              >
                <item.icon className="h-5 w-5" />
                {item.label}
              </NavLink>
            ))}
          </nav>
        </SheetContent>
      </Sheet>
      <Button 
        variant="outline" 
        size="icon" 
        className="hidden md:flex" 
        onClick={() => setIsCollapsed(prev => !prev)}
      >
        <PanelLeftClose className="h-5 w-5" />
        <span className="sr-only">Toggle sidebar</span>
      </Button>
      <h1 className="text-base font-medium">Dashboard</h1>

      {/* ------------------------------------------- */}

      <div className="w-full flex-1">
        {/* Bisa diisi dengan Breadcrumbs atau Search bar nanti */}
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="secondary" size="icon" className="rounded-full">
            <CircleUser className="h-5 w-5" />
            <span className="sr-only">Toggle user menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>My Account</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => setIsPasswordModalOpen(true)}>
                        Ganti Password
                    </DropdownMenuItem>
          <DropdownMenuItem>Support</DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleLogout}>Logout</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
    <ChangePasswordModal 
                isOpen={isPasswordModalOpen} 
                onClose={() => setIsPasswordModalOpen(false)} 
            />
    </>
  );
}