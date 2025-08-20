// File: src/components/EmployeeFormModal.tsx
import { useState, useEffect } from 'react';
import apiClient from '../services/api';
import toast from 'react-hot-toast';

import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";

interface Department { id: number; name: string; }
interface Employee {
  id: number;
  employee_nik: string;
  name: string;
  email: string;
  department_id?: number | null;
  role: string;
}

interface EmployeeFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  employee: Employee | null; // null for 'Add' mode, object for 'Edit' mode
}

export default function EmployeeFormModal({ isOpen, onClose, onSuccess, employee }: EmployeeFormModalProps) {
  const [formData, setFormData] = useState({
    employee_nik: '',
    name: '',
    email: '',
    department_id: '',
    role: 'employee',
    password: '',
  });
  const [departments, setDepartments] = useState<Department[]>([]);
  const isEditMode = employee !== null;

  useEffect(() => {
    // Fetch departments for the dropdown
    apiClient.get('/departments').then(res => setDepartments(res.data));

    // If in Edit mode, populate the form with employee data
    if (isEditMode) {
      setFormData({
        employee_nik: employee.employee_nik,
        name: employee.name,
        email: employee.email,
        department_id: employee.department_id?.toString() || '',
        role: employee.role,
        password: '', // Password is not edited here
      });
    } else {
      // Reset form for Add mode
      setFormData({
        employee_nik: '', name: '', email: '', department_id: '',
        role: 'employee', password: '',
      });
    }
  }, [employee, isOpen]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };
  
  const handleSelectChange = (name: string, value: string) => {
    setFormData({ ...formData, [name]: value });
  };

  const handleSubmit = () => {
    const payload = {
      ...formData,
      department_id: formData.department_id ? Number(formData.department_id) : null,
    };
    // In edit mode, we don't send the password
    if (isEditMode) {
      delete (payload as any).password;
    }

    const promise = isEditMode
      ? apiClient.put(`/employees/${employee.id}`, payload)
      : apiClient.post('/employees', payload);

    toast.promise(promise, {
      loading: 'Menyimpan data karyawan...',
      success: () => {
        onSuccess();
        return `Karyawan berhasil ${isEditMode ? 'diperbarui' : 'ditambahkan'}!`;
      },
      error: (err) => err.response?.data?.error || `Gagal menyimpan data.`,
    });
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit' : 'Tambah'} Karyawan</DialogTitle>
          <DialogDescription>
            {isEditMode ? 'Perbarui detail karyawan di bawah ini.' : 'Isi detail untuk karyawan baru.'}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <Label>NIK</Label><Input name="employee_nik" value={formData.employee_nik} onChange={handleChange} />
          <Label>Nama</Label><Input name="name" value={formData.name} onChange={handleChange} />
          <Label>Email</Label><Input name="email" type="email" value={formData.email} onChange={handleChange} />
          {!isEditMode && <><Label>Password</Label><Input name="password" type="password" value={formData.password} onChange={handleChange} /></>}
          <Label>Departemen</Label>
          <Select name="department_id" value={formData.department_id} onValueChange={(v) => handleSelectChange('department_id', v)}>
            <SelectTrigger><SelectValue placeholder="Pilih departemen..." /></SelectTrigger>
            <SelectContent>
              {departments.map(dept => <SelectItem key={dept.id} value={dept.id.toString()}>{dept.name}</SelectItem>)}
            </SelectContent>
          </Select>
          <Label>Role</Label>
          <Select name="role" value={formData.role} onValueChange={(v) => handleSelectChange('role', v)}>
            <SelectTrigger><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="employee">Employee</SelectItem>
              <SelectItem value="super_admin">Super Admin</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}