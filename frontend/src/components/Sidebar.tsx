// File: src/components/Sidebar.tsx
import { NavLink, Link } from 'react-router-dom';
import { Package } from 'lucide-react'; 
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../components/ui/tooltip";
import { getNavItems } from '../config/nav';

interface SidebarProps {
  isCollapsed: boolean;
}

export default function Sidebar({ isCollapsed }: SidebarProps) {
  const navItems = getNavItems();

  return (
    <aside className="hidden border-r bg-muted/40 md:block">
      <div className="flex h-full max-h-screen flex-col gap-2">
        <div className="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
          <Link to="/" className="flex items-center gap-2 font-semibold text-primary">
            <Package className="h-6 w-6" />
            {!isCollapsed && <span>IT-AMS</span>}
          </Link>
        </div>
        <div className="flex-1 overflow-auto py-2">
          <TooltipProvider delayDuration={0}>
            <nav className="flex flex-col gap-1 px-2 text-sm font-medium lg:px-4">
              {navItems.map((item) => (
                <Tooltip key={item.path}>
                  <TooltipTrigger asChild>
                    <NavLink
                      to={item.path}
                      end={item.path === '/'}
                      className={({ isActive }) =>
                        `flex rounded-lg px-3 py-2 text-muted-foreground transition-all hover:text-primary
                        ${isCollapsed ? 'h-9 w-9 justify-center' : ''}
                        ${isActive && 'bg-muted text-primary'}  items center`
                      }
                    >
                      <item.icon className="h-5 w-5 transition-all group-aria-[current=page]:text-primary" />
                      {!isCollapsed && <span className="truncate">{item.label}</span>}
                    </NavLink>
                  </TooltipTrigger>
                  {isCollapsed && <TooltipContent side="right">{item.label}</TooltipContent>}
                </Tooltip>
              ))}
            </nav>
          </TooltipProvider>
        </div>
      </div>
    </aside>
  );
}