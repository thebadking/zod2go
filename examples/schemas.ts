import { z } from "zod";

export const UserSchema = z.object({
  name: z.string().min(2).max(100),
  email: z.email(),
  age: z.number().int().min(0).max(150).optional(),
  role: z.enum(["admin", "user"]).default("user"),
  tags: z.array(z.string()).min(1).optional(),
});

export const PostSchema = z.object({
  title: z.string().min(1),
  body: z.string(),
  published: z.boolean().default(false),
});

export type User = z.infer<typeof UserSchema>;
export type Post = z.infer<typeof PostSchema>;
