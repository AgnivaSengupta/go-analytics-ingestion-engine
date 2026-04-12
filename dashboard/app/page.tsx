"use client";
import Header from "@/components/Header";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/context/AuthContext";
import Image from "next/image";
import { useRouter } from "next/navigation";
// import Link from "next/link";
import { useState } from "react";

export default function LandingPage() {
  const { login, register } = useAuth();
  const router = useRouter();

  const [isLogin, setIsLogin] = useState(false);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  // const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: Connect this to your Go API authentication endpoints
    // console.log(isLogin ? "Logging in..." : "Signing up...");
    setLoading(true);
    try {
      if (isLogin) {
        await login(email, password);
        router.push("/sites");
      } else {
        await register(name, email, password);
        router.push("/sites");
      }
    } catch (error) {
      console.log("[Error]: ", error);
      // setError(error.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <div className="fixed inset-0 w-full h-full -z-10">
        <Image
          src="/Bg-image.png"
          alt="Misty mountain"
          fill
          priority
          className="object-cover object-center -z-10"
        />
      </div>
      <main className="relative w-full min-h-[100dvh] flex flex-col overflow-hidden">
          <Header />
          <div className="max-w-7xl mx-auto mt-20 flex flex-col gap-4 text-center">
            <div className="rounded-full bg-blue-50 w-fit mx-auto py-1 px-5">
              <p className="text-sm text-zinc-900">
                A High-Throughput Analytics Pipeline
              </p>
            </div>

            <div className="flex flex-col gap-2 font-serif tracking-wide">
              <h1 className=" text-7xl">Measure everything.</h1>
              <h2 className="text-6xl text-zinc-500">Wait for nothing.</h2>
            </div>

            <p className="max-w-3xl tracking-wide text-base text-zinc-600 my-4">
              An open-source data ingestion engine engineered in Go. Decoupling
              HTTP reception from database persistence to process thousands of
              events per second with sub-10ms API latency.
            </p>

            <div className="flex items-center mx-auto">
              <Dialog>
                <DialogTrigger asChild>
                  <Button
                    size="lg"
                    className="px-10 py-6 text-xl font-serif rounded-lg cursor-pointer bg-zinc-900 text-white hover:bg-zinc-800 transition-colors"
                  >
                    Get started
                  </Button>
                </DialogTrigger>

                <DialogContent className="sm:max-w-[400px] p-8">
                  <DialogHeader className="text-left mb-4">
                    <DialogTitle className="font-serif text-3xl text-zinc-900">
                      {isLogin ? "Welcome back" : "Create an account"}
                    </DialogTitle>
                    <DialogDescription className="text-zinc-500 text-sm">
                      {isLogin
                        ? "Enter your credentials to access your dashboard."
                        : "Sign up to start tracking your events in real-time."}
                    </DialogDescription>
                  </DialogHeader>

                  <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                    {!isLogin && (
                      <div className="flex flex-col gap-2 text-left">
                        <Label htmlFor="name" className="text-zinc-700 text-sm">
                          Full Name
                        </Label>
                        <Input
                          id="name"
                          type="text"
                          placeholder="User Name"
                          value={name}
                          onChange={(e) => setName(e.target.value)}
                          required
                          className="focus-visible:ring-zinc-900 text-sm"
                        />
                      </div>
                    )}

                    <div className="flex flex-col gap-2 text-left">
                      <Label htmlFor="email" className="text-zinc-700 text-sm">
                        Email Address
                      </Label>
                      <Input
                        id="email"
                        type="email"
                        placeholder="john@example.com"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        required
                        className="focus-visible:ring-zinc-900 text-sm"
                      />
                    </div>

                    <div className="flex flex-col gap-2 text-left">
                      <Label
                        htmlFor="password"
                        className="text-zinc-700 text-sm"
                      >
                        Password
                      </Label>
                      <Input
                        id="password"
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                        className="focus-visible:ring-zinc-900 text-sm"
                      />
                    </div>

                    <Button
                      type="submit"
                      size="lg"
                      className="w-full mt-1 bg-zinc-900 text-sm py-4 text-white hover:bg-zinc-800 cursor-pointer"
                      disabled={loading}
                    >
                      {isLogin ? "Log In" : "Sign Up"}
                    </Button>
                  </form>

                  <div className="text-center mt-6 text-sm text-zinc-500">
                    {isLogin
                      ? "Don't have an account? "
                      : "Already have an account? "}
                    <button
                      type="button"
                      onClick={() => setIsLogin(!isLogin)}
                      className="text-zinc-900 font-medium underline underline-offset-4 hover:text-zinc-600 transition-colors cursor-pointer"
                    >
                      {isLogin ? "Sign up" : "Log in"}
                    </button>
                  </div>
                </DialogContent>
              </Dialog>
            </div>
          </div>
        
      </main>
    </>
  );
}
