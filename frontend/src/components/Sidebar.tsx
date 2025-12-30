// =========================
// 📁 File: src/components/Sidebar.tsx
// =========================
import { useState, useEffect, useMemo } from "react"
import { NavLink } from "react-router-dom"
import { getNavItems, type RoleType } from "@/config/nav_back"
import { useAuthContext } from "@/context/AuthContext"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { motion, AnimatePresence } from "framer-motion"
import {
  ChevronDown,
  ChevronRight,
  LogOut,
  Menu,
  X,
} from "lucide-react"

function cn(...xs: Array<string | false | undefined>) {
  return xs.filter(Boolean).join(" ")
}

const GROUP_KEY = "sidebar:lastOpenGroup"

export default function Sidebar() {
  const [collapsed, setCollapsed] = useState(false)
  const [openGroup, setOpenGroup] = useState<string>("Main") // default open
  const [isMobile, setIsMobile] = useState(false)


  const {
    role,
    delegatedRole,
    effectiveRole,
    setEffectiveRole,
    departmentId,
    logout,
  } = useAuthContext()

  const currentRole: RoleType = (effectiveRole || role || "employee") as RoleType
  
  const navItems = useMemo(() => getNavItems(currentRole), [currentRole])

  // ===== Grouping menu =====
    const grouped = {
    Main: navItems.filter((i) => i.path === "/"),
    "Assets Management": navItems.filter((i) =>
      /(asset|assets|asset-type)/i.test(i.path)
    ),
    Organization: navItems.filter((i) =>
      /(employees|departments|locations)/i.test(i.path)
    ),
    Governance: navItems.filter(
      (i) => i.path && /(cost-centers|budgets|licenses|contracts|reports)/i.test(i.path)
    ),
    Compliance: navItems.filter((i) =>
      /(audit|compliance|verification)/i.test(i.path)
    ),
    "Support & Monitoring": navItems.filter((i) =>
      /(tickets|sla|alerts|correlation)/i.test(i.path)
    ),
    Security: navItems.filter((i) =>
      /(security|anomaly|risk)/i.test(i.path)
    ),
    "Learning & Development": navItems.filter((i) =>
      /(training)/i.test(i.path)
    ),
  }


  // ===== Responsive =====
  useEffect(() => {
    const handleResize = () => {
      const mobile = window.innerWidth < 1024
      setIsMobile(mobile)
      if (mobile) setCollapsed(true)
    }
    handleResize()
    window.addEventListener("resize", handleResize)
    return () => window.removeEventListener("resize", handleResize)
  }, [])

  // ===== Remember last open group =====
  useEffect(() => {
    const saved = localStorage.getItem(GROUP_KEY)
    if (saved) setOpenGroup(saved)
  }, [])
  useEffect(() => {
    if (openGroup) localStorage.setItem(GROUP_KEY, openGroup)
  }, [openGroup])

  // ===== Shared content =====
  const Section = ({
    title,
    items,
    hasDivider,
  }: {
    title: string
    items: typeof navItems
    hasDivider?: boolean
  }) => {
    if (!items.length) return null
    const opened = openGroup === title
    return (
      <div className="px-2">
        <button
          onClick={() => setOpenGroup((prev) => (prev === title ? "" : title))}
          className={cn(
            "group flex w-full items-center justify-between rounded-md px-2 py-2",
            "text-[13px] font-semibold uppercase tracking-wide",
            "text-muted-foreground hover:text-foreground"
          )}
          aria-expanded={opened}
          aria-controls={`section-${title}`}
        >
          <span className="select-none">{title}</span>
          {opened ? (
            <ChevronDown className="h-3.5 w-3.5 opacity-70 group-hover:opacity-100" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 opacity-70 group-hover:opacity-100" />
          )}
        </button>

        <AnimatePresence initial={false}>
          {opened && (
            <motion.ul
              id={`section-${title}`}
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.25 }}
              className="mb-1 space-y-0.5"
            >
              {items.map((item) => (
                <li key={item.path}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <NavLink
                        to={item.path}
                        onClick={() => isMobile && setCollapsed(true)}
                        className={({ isActive }) =>
                          cn(
                            "relative flex items-center gap-3 rounded-md px-3 py-2 text-[0.85rem] leading-5",
                            "hover:bg-accent/60 hover:text-accent-foreground",
                            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                            isActive
                              ? "bg-accent text-accent-foreground border-l-2 border-primary font-medium"
                              : "border-l-2 border-transparent text-foreground/90"
                          )
                        }
                      >
                        <item.icon
                          className={cn(
                            "h-[18px] w-[18px] shrink-0 text-muted-foreground group-hover:text-foreground"
                          )}
                        />
                        <span className="truncate">{item.label}</span>
                      </NavLink>
                    </TooltipTrigger>
                    <TooltipContent side="right">{item.label}</TooltipContent>
                  </Tooltip>
                </li>
              ))}
            </motion.ul>
          )}
        </AnimatePresence>

        {hasDivider && (
          <div className="mx-2 my-1 border-b border-border/40" aria-hidden="true" />
        )}
      </div>
    )
  }

  const SidebarInner = (
    <>
      {/* Header brand */}
      <div className="flex items-center justify-between border-b px-3 py-2">
        <span className="font-semibold text-sm tracking-wide center">ITAM / ITSM</span>
        {isMobile && (
          <button
            onClick={() => setCollapsed(true)}
            className="rounded-md p-1 hover:bg-accent"
            aria-label="Close sidebar"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-1 scrollbar-thin">
        {Object.entries(grouped).map(([section, items], index, arr) => (
          <Section
            key={section}
            title={section}
            items={items}
            hasDivider={index !== arr.length - 1}
          />
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t p-3 text-xs">
        <p className="font-medium">
          Active: {effectiveRole?.toUpperCase() || "GUEST"}
        </p>
        {departmentId && (
          <p className="text-muted-foreground">Dept ID: {departmentId}</p>
        )}

        {delegatedRole && delegatedRole !== role && (
          <div className="mt-2">
            <label className="text-[11px] text-muted-foreground">Delegated Role</label>
            <select
              value={effectiveRole || role || ""}
              onChange={(e) => setEffectiveRole(e.target.value)}
              className="mt-1 w-full rounded-md border bg-background px-1 py-1 text-xs"
            >
              <option value={role || ""}>{role?.toUpperCase()}</option>
              <option value={delegatedRole}>{delegatedRole?.toUpperCase()}</option>
            </select>
          </div>
        )}

        <button
          onClick={logout}
          className="mt-3 w-full rounded-md px-2 py-1.5 text-left text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <div className="flex items-center gap-2">
            <LogOut className="h-4 w-4" /> Logout
          </div>
        </button>
      </div>
    </>
  )

  // Mobile: slide-in/out
  if (isMobile) {
    return (
      <>
        <AnimatePresence>
          {!collapsed && (
            <>
              <motion.div
                className="fixed inset-0 z-30 bg-black/40"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.2 }}
                onClick={() => setCollapsed(true)}
              />
              <motion.aside
                initial={{ x: -260 }}
                animate={{ x: 0 }}
                exit={{ x: -260 }}
                transition={{ type: "spring", stiffness: 260, damping: 28 }}
                className="fixed left-0 top-0 z-40 flex h-full w-64 flex-col border-r bg-background text-foreground"
                role="navigation"
                aria-label="Sidebar"
              >
                {SidebarInner}
              </motion.aside>
            </>
          )}
        </AnimatePresence>

        {/* Floating toggle */}
        {collapsed && (
          <button
            onClick={() => setCollapsed(false)}
            className="fixed left-3 top-3 z-50 rounded-md border bg-background p-2 shadow-md transition hover:bg-accent"
            aria-label="Open sidebar"
          >
            <Menu className="h-5 w-5" />
          </button>
        )}
      </>
    )
  }

  // Desktop
  return (
    <aside
      className="flex w-64 shrink-0 flex-col border-r bg-background text-foreground"
      role="navigation"
      aria-label="Sidebar"
    >
      {SidebarInner}
    </aside>
  )
}
